package workflow

import (
	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/auth115"
	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/mount"
	managedwebdav "github.com/OpenListTeam/OpenList/v4/internal/media_drive/webdav"
)

type ComponentHealth struct {
	State string `json:"state"`
	Ready bool   `json:"ready"`
}

type HealthReport struct {
	State      State           `json:"state"`
	Healthy    bool            `json:"healthy"`
	Auth       ComponentHealth `json:"auth"`
	WebDAV     ComponentHealth `json:"webdav"`
	Mount      ComponentHealth `json:"mount"`
	Diagnostic *Diagnostic     `json:"diagnostic,omitempty"`
}

func (m *Manager) Health() HealthReport {
	m.mu.Lock()
	defer m.mu.Unlock()

	report, diagnostic, allReady := m.healthLocked()
	if m.state == StateRunning && !allReady {
		m.state = StateDegraded
		m.diagnostic = diagnostic
	}
	if m.state == StateDegraded && allReady {
		m.state = StateRunning
		m.diagnostic = nil
	}
	report.State = m.state
	report.Healthy = m.state == StateRunning && allReady
	report.Diagnostic = cloneDiagnostic(m.diagnostic)
	if report.Diagnostic == nil {
		report.Diagnostic = cloneDiagnostic(diagnostic)
	}
	return report
}

func (m *Manager) healthLocked() (HealthReport, *Diagnostic, bool) {
	report := HealthReport{State: m.state}
	authStatus := auth115.StorageStatus{}
	authReady := false
	if m.auth != nil {
		authStatus = m.auth.StorageStatus()
		authReady = authDiagnostic(authStatus) == nil
	}
	report.Auth = ComponentHealth{State: "NOT_READY", Ready: authReady}
	if authReady {
		report.Auth.State = "READY"
	}

	webStatus, webErr := managedwebdav.ServiceStatus{}, error(nil)
	if m.webdav != nil {
		webStatus, webErr = m.webdav.Status()
	}
	webReady := webErr == nil && webStatus.Running
	report.WebDAV = ComponentHealth{State: string(webStatus.State), Ready: webReady}
	if webErr != nil {
		report.WebDAV.State = "UNKNOWN"
	}

	mountStatus, mountErr := mount.MountStatus{}, error(nil)
	if m.mount != nil {
		mountStatus, mountErr = m.mount.Status()
	}
	mountReady := mountErr == nil && mountStatus.Mounted
	report.Mount = ComponentHealth{State: string(mountStatus.State), Ready: mountReady}
	if mountErr != nil {
		report.Mount.State = "UNKNOWN"
	}

	var diagnostic *Diagnostic
	if !authReady {
		diagnostic = authDiagnostic(authStatus)
		if diagnostic == nil {
			diagnostic = workflowDiagnostic("AUTH_CHECK_UNAVAILABLE", "115 authorization status is unavailable", "Check the 115 storage status")
		}
	} else if diagnostic = webDAVDiagnostic(webStatus, webErr); diagnostic == nil {
		diagnostic = mountDiagnostic(mountStatus, mountErr)
	}
	return report, diagnostic, authReady && webReady && mountReady
}

func cloneDiagnostic(diagnostic *Diagnostic) *Diagnostic {
	if diagnostic == nil {
		return nil
	}
	copy := *diagnostic
	return &copy
}
