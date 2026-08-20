package webdav

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
	Load() (ManagedWebDAVProfile, error)
	Save(ManagedWebDAVProfile) error
}

type SettingProfileStore struct{}

func NewSettingProfileStore() ProfileStore {
	return SettingProfileStore{}
}

func (SettingProfileStore) Load() (ManagedWebDAVProfile, error) {
	item, err := op.GetSettingItemByKey(ProfileSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DefaultProfile(), nil
	}
	if err != nil {
		return ManagedWebDAVProfile{}, err
	}
	var profile ManagedWebDAVProfile
	if err := json.Unmarshal([]byte(item.Value), &profile); err != nil {
		return ManagedWebDAVProfile{}, err
	}
	return normalizeProfile(profile), nil
}

func (SettingProfileStore) Save(profile ManagedWebDAVProfile) error {
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
	mu      sync.Mutex
	store   ProfileStore
	profile ManagedWebDAVProfile
	loaded  bool
	service *Service
}

func NewManager(store ProfileStore) *Manager {
	return &Manager{
		store:   store,
		profile: DefaultProfile(),
		service: NewService(),
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

func (m *Manager) Profile() (ProfileView, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return ProfileView{}, err
	}
	return m.profile.View(), nil
}

func (m *Manager) UpdateProfile(update ProfileUpdate) (ProfileView, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return ProfileView{}, err
	}
	if m.service.Status().Running {
		return ProfileView{}, ErrServiceRunning
	}

	profile := m.profile
	if update.Enabled != nil {
		profile.Enabled = *update.Enabled
	}
	if update.BindAddress != "" {
		profile.BindAddress = update.BindAddress
	}
	if update.Port != nil {
		profile.Port = *update.Port
	}
	if update.Username != "" {
		profile.Username = update.Username
	}
	if update.AllowLocalhostOnly != nil {
		profile.AllowLocalhostOnly = *update.AllowLocalhostOnly
	}
	if update.Password != "" {
		hash, err := hashPassword(update.Password)
		if err != nil {
			return ProfileView{}, err
		}
		profile.PasswordHash = hash
	}
	profile = normalizeProfile(profile)
	profile.UpdatedAt = time.Now()
	if err := validateProfile(profile); err != nil {
		return ProfileView{}, err
	}
	if m.store != nil {
		if err := m.store.Save(profile); err != nil {
			return ProfileView{}, err
		}
	}
	m.profile = profile
	return profile.View(), nil
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return err
	}
	if !m.profile.Enabled {
		return ErrProfileDisabled
	}
	return m.service.Start(m.profile)
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.service.Stop()
}

func (m *Manager) Status() (ServiceStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return ServiceStatus{}, err
	}
	status := m.service.Status()
	status.Enabled = m.profile.Enabled
	return status, nil
}
