package mount

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryProfileStore struct {
	mu      sync.Mutex
	profile MountProfile
	value   string
}

func (s *memoryProfileStore) Load() (MountProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.profile.DriveLetter == "" {
		return DefaultProfile(), nil
	}
	return s.profile, nil
}

func (s *memoryProfileStore) Save(profile MountProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profile = profile
	value, err := json.Marshal(profile)
	if err == nil {
		s.value = string(value)
	}
	return err
}

func TestMountLifecycle(t *testing.T) {
	backend := &mockBackend{}
	manager := NewManagerWithBackend(nil, backend)
	profile := testProfile()
	if err := manager.UpdateProfile(profile); err != nil {
		t.Fatal(err)
	}
	if err := manager.Mount(); err != nil {
		t.Fatal(err)
	}
	defer manager.Unmount()

	status, err := manager.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.State != StateMounted || !status.Mounted || status.DriveLetter != "R:" {
		t.Fatalf("status = %+v", status)
	}
}

func TestUnmount(t *testing.T) {
	backend := &mockBackend{}
	manager := NewManagerWithBackend(nil, backend)
	if err := manager.UpdateProfile(testProfile()); err != nil {
		t.Fatal(err)
	}
	if err := manager.Mount(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Unmount(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Unmount(); err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.State != StateUnmounted || status.Mounted {
		t.Fatalf("status = %+v", status)
	}
	if backend.last() == nil || !backend.last().wasUnmounted() {
		t.Fatal("backend handle was not unmounted")
	}
}

func TestReconnect(t *testing.T) {
	first := newMockHandle()
	second := newMockHandle()
	backend := &mockBackend{handles: []*mockHandle{first, second}}
	manager := NewManagerWithBackend(nil, backend)
	profile := testProfile()
	profile.AutoReconnect = true
	if err := manager.UpdateProfile(profile); err != nil {
		t.Fatal(err)
	}
	if err := manager.Mount(); err != nil {
		t.Fatal(err)
	}
	first.fail(errors.New("connection lost"))
	defer manager.Unmount()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		status, err := manager.Status()
		if err != nil {
			t.Fatal(err)
		}
		if status.State == StateMounted && backend.count() == 2 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("mount did not reconnect: status=%+v mounts=%d", mustStatus(t, manager), backend.count())
}

func TestInvalidDriveLetter(t *testing.T) {
	manager := NewManagerWithBackend(nil, &mockBackend{})
	profile := testProfile()
	profile.DriveLetter = "RR:"
	if err := manager.UpdateProfile(profile); !errors.Is(err, ErrInvalidDriveLetter) {
		t.Fatalf("UpdateProfile error = %v, want %v", err, ErrInvalidDriveLetter)
	}
}

func TestPublicWebDAVRejected(t *testing.T) {
	manager := NewManagerWithBackend(nil, &mockBackend{})
	profile := testProfile()
	profile.WebDAVURL = "http://example.com:19080"
	if err := manager.UpdateProfile(profile); !errors.Is(err, ErrPublicWebDAV) {
		t.Fatalf("UpdateProfile error = %v, want %v", err, ErrPublicWebDAV)
	}
}

func TestMountCredentialsNotPersisted(t *testing.T) {
	store := &memoryProfileStore{}
	manager := NewManagerWithBackend(store, &mockBackend{})
	if err := manager.UpdateProfile(testProfile()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(store.value, "secret") || strings.Contains(store.value, "media") {
		t.Fatalf("runtime credentials persisted: %s", store.value)
	}
}

func testProfile() MountProfile {
	profile := DefaultProfile()
	profile.Enabled = true
	profile.Username = "media"
	profile.Password = "secret"
	return profile
}

func mustStatus(t *testing.T, manager *Manager) MountStatus {
	t.Helper()
	status, err := manager.Status()
	if err != nil {
		t.Fatal(err)
	}
	return status
}

type mockBackend struct {
	mu      sync.Mutex
	handles []*mockHandle
	used    []*mockHandle
	mounts  int
}

func (b *mockBackend) Mount(MountProfile) (MountHandle, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.mounts++
	if len(b.handles) == 0 {
		handle := newMockHandle()
		b.handles = append(b.handles, handle)
	}
	handle := b.handles[0]
	b.handles = b.handles[1:]
	b.used = append(b.used, handle)
	return handle, nil
}

func (b *mockBackend) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.mounts
}

func (b *mockBackend) last() *mockHandle {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.used) == 0 {
		return nil
	}
	return b.used[len(b.used)-1]
}

type mockHandle struct {
	done       chan error
	unmounted  chan struct{}
	stopOnce   sync.Once
	finishOnce sync.Once
}

func newMockHandle() *mockHandle {
	return &mockHandle{
		done:      make(chan error, 1),
		unmounted: make(chan struct{}),
	}
}

func (h *mockHandle) Wait() error {
	return <-h.done
}

func (h *mockHandle) Unmount() error {
	h.stopOnce.Do(func() {
		close(h.unmounted)
		h.finish(nil)
	})
	return nil
}

func (h *mockHandle) fail(err error) {
	h.finish(err)
}

func (h *mockHandle) finish(err error) {
	h.finishOnce.Do(func() {
		h.done <- err
	})
}

func (h *mockHandle) wasUnmounted() bool {
	select {
	case <-h.unmounted:
		return true
	default:
		return false
	}
}
