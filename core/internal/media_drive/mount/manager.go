package mount

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"gorm.io/gorm"
)

type ProfileStore interface {
	Load() (MountProfile, error)
	Save(MountProfile) error
}

type SettingProfileStore struct{}

func NewSettingProfileStore() ProfileStore {
	return SettingProfileStore{}
}

func (SettingProfileStore) Load() (MountProfile, error) {
	item, err := op.GetSettingItemByKey(ProfileSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DefaultProfile(), nil
	}
	if err != nil {
		return MountProfile{}, err
	}
	var profile MountProfile
	if err := json.Unmarshal([]byte(item.Value), &profile); err != nil {
		return MountProfile{}, err
	}
	return normalizeProfile(profile), nil
}

func (SettingProfileStore) Save(profile MountProfile) error {
	value, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	return op.SaveSettingItem(&model.SettingItem{
		Key:   ProfileSettingKey,
		Value: string(value),
		Type:  "string",
		Group: model.SINGLE,
		Flag:  model.PRIVATE,
	})
}

type Manager struct {
	mu         sync.Mutex
	store      ProfileStore
	backend    MountBackend
	profile    MountProfile
	loaded     bool
	state      MountState
	current    MountHandle
	generation uint64
	lastError  error
}

func NewManager(store ProfileStore) *Manager {
	return NewManagerWithBackend(store, newWinFSPBackend())
}

func NewManagerWithBackend(store ProfileStore, backend MountBackend) *Manager {
	return &Manager{
		store:   store,
		backend: backend,
		profile: DefaultProfile(),
		state:   StateUnmounted,
	}
}

func (m *Manager) loadLocked() error {
	if m.loaded {
		return nil
	}
	if m.store != nil {
		profile, err := m.store.Load()
		if err != nil {
			return err
		}
		m.profile = normalizeProfile(profile)
	}
	m.loaded = true
	return nil
}

func (m *Manager) Profile() (MountProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return MountProfile{}, err
	}
	return m.profile, nil
}

func (m *Manager) UpdateProfile(profile MountProfile) error {
	profile = normalizeProfile(profile)
	if err := validateProfile(profile); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return err
	}
	if m.state != StateUnmounted && m.state != StateFailed {
		return ErrMountRunning
	}
	if profile.Username == "" {
		profile.Username = m.profile.Username
	}
	if profile.Password == "" {
		profile.Password = m.profile.Password
	}
	if m.store != nil {
		if err := m.store.Save(profile); err != nil {
			return err
		}
	}
	m.profile = profile
	m.lastError = nil
	return nil
}

func (m *Manager) Mount() error {
	m.mu.Lock()
	if err := m.loadLocked(); err != nil {
		m.mu.Unlock()
		return err
	}
	profile := m.profile
	if !profile.Enabled {
		m.mu.Unlock()
		return ErrMountDisabled
	}
	if err := validateProfile(profile); err != nil {
		m.mu.Unlock()
		return err
	}
	if profile.Username == "" || profile.Password == "" {
		m.mu.Unlock()
		return ErrMountCredentials
	}
	if m.state == StateMounted || m.state == StateMounting || m.state == StateUnmounting {
		m.mu.Unlock()
		return ErrMountRunning
	}
	m.state = StateMounting
	m.lastError = nil
	m.generation++
	generation := m.generation
	m.mu.Unlock()

	return m.mountAttempt(profile, generation)
}

func (m *Manager) mountAttempt(profile MountProfile, generation uint64) error {
	handle, err := m.backend.Mount(profile)
	m.mu.Lock()
	if err != nil {
		if m.generation == generation && m.state == StateMounting {
			m.state = StateFailed
			m.lastError = err
		}
		m.mu.Unlock()
		return err
	}
	if m.generation != generation || m.state != StateMounting {
		m.mu.Unlock()
		_ = handle.Unmount()
		return ErrMountCanceled
	}
	m.current = handle
	m.state = StateMounted
	m.mu.Unlock()

	go m.watch(handle, profile, generation)
	return nil
}

func (m *Manager) watch(handle MountHandle, profile MountProfile, generation uint64) {
	err := handle.Wait()
	m.mu.Lock()
	if m.generation != generation || m.current != handle {
		m.mu.Unlock()
		return
	}
	m.current = nil
	if m.state == StateUnmounting || m.state == StateUnmounted {
		m.state = StateUnmounted
		m.lastError = nil
		m.mu.Unlock()
		return
	}
	if err == nil {
		err = ErrMountFailed
	}
	m.lastError = err
	if !profile.AutoReconnect {
		m.state = StateFailed
		m.mu.Unlock()
		return
	}
	m.state = StateMounting
	m.mu.Unlock()

	go m.reconnect(profile, generation)
}

func (m *Manager) reconnect(profile MountProfile, generation uint64) {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	<-timer.C
	m.mu.Lock()
	if m.generation != generation || m.state != StateMounting {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
	_ = m.mountAttempt(profile, generation)
}

func (m *Manager) Unmount() error {
	m.mu.Lock()
	if err := m.loadLocked(); err != nil {
		m.mu.Unlock()
		return err
	}
	if m.state == StateUnmounted {
		m.mu.Unlock()
		return nil
	}
	m.generation++
	m.state = StateUnmounting
	handle := m.current
	m.mu.Unlock()

	var err error
	if handle != nil {
		err = handle.Unmount()
	}
	m.mu.Lock()
	m.current = nil
	m.state = StateUnmounted
	m.lastError = nil
	m.mu.Unlock()
	return err
}

func (m *Manager) Status() (MountStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return MountStatus{}, err
	}
	status := MountStatus{
		DriveLetter:   m.profile.DriveLetter,
		WebDAVURL:     m.profile.WebDAVURL,
		Enabled:       m.profile.Enabled,
		AutoReconnect: m.profile.AutoReconnect,
		Mounted:       m.state == StateMounted,
		State:         m.state,
	}
	if m.lastError != nil {
		status.Error = m.lastError.Error()
	}
	return status, nil
}
