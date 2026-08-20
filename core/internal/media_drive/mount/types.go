package mount

import (
	"errors"
	"net/url"
	"strings"
)

const ProfileSettingKey = "media_drive_mount_profile"

type MountState string

const (
	StateUnmounted  MountState = "UNMOUNTED"
	StateMounting   MountState = "MOUNTING"
	StateMounted    MountState = "MOUNTED"
	StateUnmounting MountState = "UNMOUNTING"
	StateFailed     MountState = "FAILED"
)

var (
	ErrInvalidDriveLetter = errors.New("INVALID_DRIVE_LETTER")
	ErrInvalidWebDAVURL   = errors.New("INVALID_WEBDAV_URL")
	ErrPublicWebDAV       = errors.New("PUBLIC_WEBDAV_FORBIDDEN")
	ErrMountDisabled      = errors.New("MOUNT_DISABLED")
	ErrMountRunning       = errors.New("MOUNT_RUNNING")
	ErrMountCanceled      = errors.New("MOUNT_CANCELED")
	ErrMountFailed        = errors.New("MOUNT_FAILED")
	ErrMountCredentials   = errors.New("MOUNT_CREDENTIALS_REQUIRED")
	ErrWinFSPUnavailable  = errors.New("WINFSP_UNAVAILABLE")
	ErrUnmountFailed      = errors.New("UNMOUNT_FAILED")
)

type MountProfile struct {
	DriveLetter   string `json:"drive_letter"`
	WebDAVURL     string `json:"webdav_url"`
	Enabled       bool   `json:"enabled"`
	AutoReconnect bool   `json:"auto_reconnect"`

	// Credentials are runtime-only. They are required to connect to the
	// managed WebDAV service but are never written to the profile setting.
	Username string `json:"-"`
	Password string `json:"-"`
}

type MountStatus struct {
	DriveLetter   string     `json:"drive_letter"`
	WebDAVURL     string     `json:"webdav_url"`
	Enabled       bool       `json:"enabled"`
	AutoReconnect bool       `json:"auto_reconnect"`
	Mounted       bool       `json:"mounted"`
	State         MountState `json:"state"`
	Error         string     `json:"error,omitempty"`
}

func DefaultProfile() MountProfile {
	return MountProfile{
		DriveLetter:   "R:",
		WebDAVURL:     "http://127.0.0.1:19080",
		AutoReconnect: true,
	}
}

func normalizeProfile(profile MountProfile) MountProfile {
	defaults := DefaultProfile()
	if strings.TrimSpace(profile.DriveLetter) == "" {
		profile.DriveLetter = defaults.DriveLetter
	}
	profile.DriveLetter = strings.ToUpper(strings.TrimSpace(profile.DriveLetter))
	if strings.TrimSpace(profile.WebDAVURL) == "" {
		profile.WebDAVURL = defaults.WebDAVURL
	}
	return profile
}

func validateProfile(profile MountProfile) error {
	if len(profile.DriveLetter) != 2 || profile.DriveLetter[1] != ':' ||
		(profile.DriveLetter[0] < 'A' || profile.DriveLetter[0] > 'Z') {
		return ErrInvalidDriveLetter
	}
	parsed, err := url.Parse(profile.WebDAVURL)
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() == "" || parsed.User != nil {
		return ErrInvalidWebDAVURL
	}
	if parsed.Port() == "" {
		return ErrInvalidWebDAVURL
	}
	if parsed.Hostname() != "127.0.0.1" {
		return ErrPublicWebDAV
	}
	return nil
}
