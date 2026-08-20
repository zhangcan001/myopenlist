package workflow

import (
	"errors"

	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/auth115"
	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/mount"
	managedwebdav "github.com/OpenListTeam/OpenList/v4/internal/media_drive/webdav"
)

const (
	module115    = "115"
	moduleWebDAV = "WEBDAV"
	moduleMount  = "WINFSP_MOUNT"
)

func authDiagnostic(status auth115.StorageStatus) *Diagnostic {
	if status.Persistence.State == "FAILED" {
		return &Diagnostic{
			Module:     module115,
			Code:       "TOKEN_PERSISTENCE_FAILED",
			Reason:     "115 token persistence is not healthy",
			Suggestion: "Retry 115 token persistence before starting the media drive",
		}
	}
	if !status.Connected || status.MountPath != "/115" {
		return &Diagnostic{
			Module:     module115,
			Code:       "AUTH_REQUIRED",
			Reason:     "115 authorization is not ready",
			Suggestion: "Complete 115 authorization or import a valid token pair",
		}
	}
	return nil
}

func webDAVDiagnostic(status managedwebdav.ServiceStatus, err error) *Diagnostic {
	if err != nil {
		return webDAVErrorDiagnostic(err)
	}
	if !status.Running {
		return &Diagnostic{
			Module:     moduleWebDAV,
			Code:       "WEBDAV_NOT_RUNNING",
			Reason:     "managed localhost WebDAV is not running",
			Suggestion: "Enable the managed WebDAV profile and check its password and port",
		}
	}
	return nil
}

func webDAVErrorDiagnostic(err error) *Diagnostic {
	code := "WEBDAV_START_FAILED"
	suggestion := "Check the managed WebDAV profile and localhost port"
	switch {
	case errors.Is(err, managedwebdav.ErrPasswordNotConfigured):
		code = "WEBDAV_PASSWORD_REQUIRED"
		suggestion = "Configure a managed WebDAV password before starting the workflow"
	case errors.Is(err, managedwebdav.ErrProfileDisabled):
		code = "WEBDAV_PROFILE_DISABLED"
		suggestion = "Enable the managed WebDAV profile"
	case errors.Is(err, managedwebdav.ErrPortConflict):
		code = "WEBDAV_PORT_CONFLICT"
		suggestion = "Stop the process using the localhost WebDAV port or choose another port"
	}
	return &Diagnostic{
		Module:     moduleWebDAV,
		Code:       code,
		Reason:     "managed localhost WebDAV could not be started",
		Suggestion: suggestion,
	}
}

func mountDiagnostic(status mount.MountStatus, err error) *Diagnostic {
	if err != nil {
		return mountErrorDiagnostic(err)
	}
	if !status.Mounted {
		return &Diagnostic{
			Module:     moduleMount,
			Code:       "MOUNT_NOT_MOUNTED",
			Reason:     "the Windows media drive is not mounted",
			Suggestion: "Install WinFsp and ensure the configured drive letter is unused",
		}
	}
	return nil
}

func mountErrorDiagnostic(err error) *Diagnostic {
	code := "MOUNT_FAILED"
	suggestion := "Check WinFsp, drive-letter availability, and the localhost WebDAV service"
	switch {
	case errors.Is(err, mount.ErrWinFSPUnavailable):
		code = "WINFSP_UNAVAILABLE"
		suggestion = "Install and start WinFsp for the current Windows architecture"
	case errors.Is(err, mount.ErrInvalidDriveLetter):
		code = "INVALID_DRIVE_LETTER"
		suggestion = "Choose one unused Windows drive letter such as R:"
	case errors.Is(err, mount.ErrPublicWebDAV), errors.Is(err, mount.ErrInvalidWebDAVURL):
		code = "INVALID_LOCALHOST_WEBDAV"
		suggestion = "Use the managed WebDAV URL bound to 127.0.0.1"
	case errors.Is(err, mount.ErrMountCredentials):
		code = "MOUNT_CREDENTIALS_REQUIRED"
		suggestion = "Provide the managed WebDAV password when starting the workflow"
	}
	return &Diagnostic{
		Module:     moduleMount,
		Code:       code,
		Reason:     "the Windows media drive could not be mounted",
		Suggestion: suggestion,
	}
}

func workflowDiagnostic(code, reason, suggestion string) *Diagnostic {
	return &Diagnostic{Module: "WORKFLOW", Code: code, Reason: reason, Suggestion: suggestion}
}
