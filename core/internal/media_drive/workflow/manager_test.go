package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/auth115"
	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/mount"
	managedwebdav "github.com/OpenListTeam/OpenList/v4/internal/media_drive/webdav"
)

func TestWorkflowStartOrder(t *testing.T) {
	events := []string{}
	auth := &fakeAuth{events: &events, status: readyAuthStatus()}
	webdav := &fakeWebDAV{events: &events, profile: readyWebDAVProfile()}
	mountManager := &fakeMount{events: &events}
	manager := NewManager(auth, webdav, mountManager)

	if err := manager.StartWorkflow(context.Background(), StartOptions{WebDAVPassword: "secret"}); err != nil {
		t.Fatal(err)
	}
	if !before(events, "auth", "webdav-start") || !before(events, "webdav-start", "mount") {
		t.Fatalf("start order = %v", events)
	}
	if got := manager.Status(); got.State != StateRunning || !got.Running {
		t.Fatalf("status = %+v", got)
	}
}

func TestWorkflowFailureRecovery(t *testing.T) {
	events := []string{}
	auth := &fakeAuth{events: &events, status: readyAuthStatus()}
	webdav := &fakeWebDAV{events: &events, profile: readyWebDAVProfile()}
	mountManager := &fakeMount{events: &events, mountErr: errors.New("SUPER_SECRET_ACCESS SUPER_SECRET_REFRESH")}
	manager := NewManager(auth, webdav, mountManager)

	if err := manager.StartWorkflow(context.Background(), StartOptions{WebDAVPassword: "secret"}); !errors.Is(err, ErrWorkflowFailed) {
		t.Fatalf("start error = %v", err)
	}
	status := manager.Status()
	if status.State != StateFailed || status.Diagnostic == nil || status.Diagnostic.Module != moduleMount {
		t.Fatalf("status = %+v", status)
	}
	if !webdav.stopped {
		t.Fatal("workflow did not stop WebDAV after mount failure")
	}
	if containsAny(status.Diagnostic.Reason, "SUPER_SECRET") || containsAny(status.Diagnostic.Suggestion, "SUPER_SECRET") {
		t.Fatalf("diagnostic leaked secret: %+v", status.Diagnostic)
	}
}

func TestHealthReport(t *testing.T) {
	events := []string{}
	mountManager := &fakeMount{events: &events}
	manager := NewManager(
		&fakeAuth{events: &events, status: readyAuthStatus()},
		&fakeWebDAV{events: &events, profile: readyWebDAVProfile()},
		mountManager,
	)
	if err := manager.StartWorkflow(context.Background(), StartOptions{WebDAVPassword: "secret"}); err != nil {
		t.Fatal(err)
	}
	report := manager.Health()
	if !report.Healthy || report.State != StateRunning || !report.Auth.Ready || !report.WebDAV.Ready || !report.Mount.Ready {
		t.Fatalf("health = %+v", report)
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if containsAny(string(encoded), "secret", "access_token", "refresh_token") {
		t.Fatalf("health leaked credentials: %s", encoded)
	}
	mountManager.status.State = mount.StateFailed
	mountManager.status.Mounted = false
	degraded := manager.Health()
	if degraded.State != StateDegraded || degraded.Healthy || degraded.Diagnostic == nil || degraded.Diagnostic.Module != moduleMount {
		t.Fatalf("degraded health = %+v", degraded)
	}
}

func TestNoTokenLeak(t *testing.T) {
	events := []string{}
	manager := NewManager(
		&fakeAuth{events: &events, status: auth115.StorageStatus{}},
		&fakeWebDAV{events: &events, profile: readyWebDAVProfile()},
		&fakeMount{events: &events},
	)
	if err := manager.StartWorkflow(context.Background(), StartOptions{WebDAVPassword: "SUPER_SECRET_REFRESH"}); !errors.Is(err, ErrWorkflowFailed) {
		t.Fatalf("start error = %v", err)
	}
	statusBytes, err := json.Marshal(manager.Status())
	if err != nil {
		t.Fatal(err)
	}
	healthBytes, err := json.Marshal(manager.Health())
	if err != nil {
		t.Fatal(err)
	}
	output := string(statusBytes) + string(healthBytes) + manager.Status().Diagnostic.Reason
	if containsAny(output, "SUPER_SECRET_REFRESH", "access_token", "refresh_token") {
		t.Fatalf("workflow leaked token material: %s", output)
	}
}

func readyAuthStatus() auth115.StorageStatus {
	return auth115.StorageStatus{StorageID: 1, MountPath: "/115", Connected: true, Persistence: auth115.TokenPersistenceStatus{State: "CLEAN"}}
}

func readyWebDAVProfile() managedwebdav.ProfileView {
	return managedwebdav.ProfileView{Username: "media", PasswordConfigured: true}
}

func before(events []string, first, second string) bool {
	firstIndex, secondIndex := -1, -1
	for index, event := range events {
		if event == first && firstIndex == -1 {
			firstIndex = index
		}
		if event == second && secondIndex == -1 {
			secondIndex = index
		}
	}
	return firstIndex >= 0 && secondIndex >= 0 && firstIndex < secondIndex
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

type fakeAuth struct {
	events *[]string
	status auth115.StorageStatus
}

func (f *fakeAuth) StorageStatus() auth115.StorageStatus {
	*f.events = append(*f.events, "auth")
	return f.status
}

type fakeWebDAV struct {
	events  *[]string
	profile managedwebdav.ProfileView
	status  managedwebdav.ServiceStatus
	stopped bool
}

func (f *fakeWebDAV) Profile() (managedwebdav.ProfileView, error) {
	*f.events = append(*f.events, "webdav-profile")
	return f.profile, nil
}

func (f *fakeWebDAV) UpdateProfile(update managedwebdav.ProfileUpdate) (managedwebdav.ProfileView, error) {
	*f.events = append(*f.events, "webdav-update")
	if update.Username != "" {
		f.profile.Username = update.Username
	}
	if update.Password != "" {
		f.profile.PasswordConfigured = true
	}
	if update.Enabled != nil {
		f.profile.Enabled = *update.Enabled
	}
	return f.profile, nil
}

func (f *fakeWebDAV) Start() error {
	*f.events = append(*f.events, "webdav-start")
	f.status = managedwebdav.ServiceStatus{Running: true, Address: "127.0.0.1:19080", State: managedwebdav.StateRunning}
	return nil
}

func (f *fakeWebDAV) Stop() error {
	*f.events = append(*f.events, "webdav-stop")
	f.stopped = true
	f.status = managedwebdav.ServiceStatus{State: managedwebdav.StateStopped}
	return nil
}

func (f *fakeWebDAV) Status() (managedwebdav.ServiceStatus, error) {
	*f.events = append(*f.events, "webdav-status")
	return f.status, nil
}

type fakeMount struct {
	events   *[]string
	profile  mount.MountProfile
	status   mount.MountStatus
	mountErr error
}

func (f *fakeMount) Profile() (mount.MountProfile, error) {
	*f.events = append(*f.events, "mount-profile")
	if f.profile.DriveLetter == "" {
		f.profile = mount.DefaultProfile()
	}
	return f.profile, nil
}

func (f *fakeMount) UpdateProfile(profile mount.MountProfile) error {
	*f.events = append(*f.events, "mount-update")
	f.profile = profile
	return nil
}

func (f *fakeMount) Mount() error {
	*f.events = append(*f.events, "mount")
	if f.mountErr != nil {
		return f.mountErr
	}
	f.status = mount.MountStatus{DriveLetter: f.profile.DriveLetter, WebDAVURL: f.profile.WebDAVURL, Mounted: true, State: mount.StateMounted}
	return nil
}

func (f *fakeMount) Unmount() error {
	*f.events = append(*f.events, "unmount")
	f.status.Mounted = false
	f.status.State = mount.StateUnmounted
	return nil
}

func (f *fakeMount) Status() (mount.MountStatus, error) {
	*f.events = append(*f.events, "mount-status")
	return f.status, nil
}
