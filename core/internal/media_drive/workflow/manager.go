package workflow

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/auth115"
	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/mount"
	managedwebdav "github.com/OpenListTeam/OpenList/v4/internal/media_drive/webdav"
)

type AuthSource interface {
	StorageStatus() auth115.StorageStatus
}

type WebDAVController interface {
	Profile() (managedwebdav.ProfileView, error)
	UpdateProfile(managedwebdav.ProfileUpdate) (managedwebdav.ProfileView, error)
	Start() error
	Stop() error
	Status() (managedwebdav.ServiceStatus, error)
}

type MountController interface {
	Profile() (mount.MountProfile, error)
	UpdateProfile(mount.MountProfile) error
	Mount() error
	Unmount() error
	Status() (mount.MountStatus, error)
}

type Manager struct {
	// ponytail: one workflow lock keeps start/stop/health deterministic; split
	// locks only if workflow calls become a measurable contention point.
	mu         sync.Mutex
	auth       AuthSource
	webdav     WebDAVController
	mount      MountController
	state      State
	diagnostic *Diagnostic
}

func NewManager(auth AuthSource, webdav WebDAVController, mount MountController) *Manager {
	return &Manager{auth: auth, webdav: webdav, mount: mount, state: StateInit}
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Status{State: m.state, Running: m.state == StateRunning, Diagnostic: cloneDiagnostic(m.diagnostic)}
}

func (m *Manager) StartWorkflow(ctx context.Context, options StartOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == StateRunning {
		return nil
	}
	if m.state == StateChecking || m.state == StateStarting || m.state == StateStopping {
		return ErrWorkflowRunning
	}
	m.state = StateChecking
	m.diagnostic = nil

	if err := ctx.Err(); err != nil {
		return m.failLocked(workflowDiagnostic("WORKFLOW_CANCELED", "workflow start was canceled", "Start the workflow again"))
	}
	if m.auth == nil || m.webdav == nil || m.mount == nil {
		return m.failLocked(workflowDiagnostic("DEPENDENCY_UNAVAILABLE", "a media-drive dependency is not configured", "Restart the Core service and check the media-drive configuration"))
	}
	if diagnostic := authDiagnostic(m.auth.StorageStatus()); diagnostic != nil {
		return m.failLocked(diagnostic)
	}
	m.state = StateReady

	webProfile, err := m.webdav.Profile()
	if err != nil {
		return m.failLocked(webDAVErrorDiagnostic(err))
	}
	webStatus, err := m.webdav.Status()
	if err != nil {
		return m.failLocked(webDAVErrorDiagnostic(err))
	}

	m.state = StateStarting
	webDAVStarted := false
	if !webStatus.Running {
		enabled := true
		update := managedwebdav.ProfileUpdate{Enabled: &enabled}
		if strings.TrimSpace(options.WebDAVUsername) != "" {
			update.Username = strings.TrimSpace(options.WebDAVUsername)
		}
		if options.WebDAVPassword != "" {
			update.Password = options.WebDAVPassword
		}
		webProfile, err = m.webdav.UpdateProfile(update)
		if err != nil {
			return m.failLocked(webDAVErrorDiagnostic(err))
		}
		if err = m.webdav.Start(); err != nil {
			return m.failLocked(webDAVErrorDiagnostic(err))
		}
		webDAVStarted = true
		webStatus, err = m.webdav.Status()
		if diagnostic := webDAVDiagnostic(webStatus, err); diagnostic != nil {
			if webDAVStarted {
				_ = m.webdav.Stop()
			}
			return m.failLocked(diagnostic)
		}
	}

	mountStatus, err := m.mount.Status()
	if err != nil {
		if webDAVStarted {
			_ = m.webdav.Stop()
		}
		return m.failLocked(mountDiagnostic(mountStatus, err))
	}
	if mountStatus.Mounted {
		m.state = StateRunning
		return nil
	}
	if mountStatus.State == mount.StateMounting || mountStatus.State == mount.StateUnmounting {
		if webDAVStarted {
			_ = m.webdav.Stop()
		}
		return m.failLocked(workflowDiagnostic("MOUNT_BUSY", "the Windows media drive is changing state", "Wait for the current mount operation to finish"))
	}

	mountProfile, err := m.mount.Profile()
	if err != nil {
		if webDAVStarted {
			_ = m.webdav.Stop()
		}
		return m.failLocked(mountDiagnostic(mountStatus, err))
	}
	mountProfile.Enabled = true
	mountProfile.Username = webProfile.Username
	if options.WebDAVPassword != "" {
		mountProfile.Password = options.WebDAVPassword
	}
	mountProfile.WebDAVURL, err = localhostWebDAVURL(webStatus.Address)
	if err != nil {
		if webDAVStarted {
			_ = m.webdav.Stop()
		}
		return m.failLocked(workflowDiagnostic("WEBDAV_ADDRESS_INVALID", "managed WebDAV did not report a localhost address", "Check the managed WebDAV listener configuration"))
	}
	if err = m.mount.UpdateProfile(mountProfile); err != nil {
		if webDAVStarted {
			_ = m.webdav.Stop()
		}
		return m.failLocked(mountDiagnostic(mountStatus, err))
	}
	if err := ctx.Err(); err != nil {
		if webDAVStarted {
			_ = m.webdav.Stop()
		}
		return m.failLocked(workflowDiagnostic("WORKFLOW_CANCELED", "workflow start was canceled", "Start the workflow again"))
	}
	if err := m.mount.Mount(); err != nil {
		if webDAVStarted {
			_ = m.webdav.Stop()
		}
		return m.failLocked(mountErrorDiagnostic(err))
	}
	mounted, err := m.mount.Status()
	if diagnostic := mountDiagnostic(mounted, err); diagnostic != nil {
		if webDAVStarted {
			_ = m.webdav.Stop()
		}
		return m.failLocked(diagnostic)
	}
	m.diagnostic = nil
	m.state = StateRunning
	return nil
}

func (m *Manager) StopWorkflow(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == StateStopping {
		return ErrWorkflowRunning
	}
	m.state = StateStopping
	var firstErr error
	if m.mount != nil {
		firstErr = m.mount.Unmount()
	}
	if m.webdav != nil {
		if err := m.webdav.Stop(); firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		m.state = StateFailed
		if errors.Is(firstErr, mount.ErrUnmountFailed) {
			m.diagnostic = mountErrorDiagnostic(firstErr)
		} else {
			m.diagnostic = webDAVErrorDiagnostic(firstErr)
		}
		return ErrWorkflowFailed
	}
	m.state = StateReady
	m.diagnostic = nil
	return nil
}

func (m *Manager) failLocked(diagnostic *Diagnostic) error {
	m.state = StateFailed
	m.diagnostic = cloneDiagnostic(diagnostic)
	return ErrWorkflowFailed
}

func localhostWebDAVURL(address string) (string, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil || port == "" {
		return "", errors.New("invalid localhost address")
	}
	return "http://127.0.0.1:" + port, nil
}
