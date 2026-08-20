package webdav

import (
	"errors"
	"time"
)

const ProfileSettingKey = "media_drive_webdav_profile"

type ServiceState string

const (
	StateStopped  ServiceState = "STOPPED"
	StateStarting ServiceState = "STARTING"
	StateRunning  ServiceState = "RUNNING"
	StateStopping ServiceState = "STOPPING"
	StateFailed   ServiceState = "FAILED"
)

var (
	ErrInvalidProfile        = errors.New("INVALID_PROFILE")
	ErrInvalidPort           = errors.New("INVALID_PORT")
	ErrLocalhostOnlyRequired = errors.New("LOCALHOST_ONLY_REQUIRED")
	ErrPasswordNotConfigured = errors.New("PASSWORD_NOT_CONFIGURED")
	ErrProfileDisabled       = errors.New("PROFILE_DISABLED")
	ErrServiceRunning        = errors.New("SERVICE_RUNNING")
	ErrPortConflict          = errors.New("PORT_CONFLICT")
)

type ManagedWebDAVProfile struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Enabled            bool      `json:"enabled"`
	BindAddress        string    `json:"bind_address"`
	Port               int       `json:"port"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"password_hash"`
	AllowLocalhostOnly bool      `json:"allow_localhost_only"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ProfileView struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Enabled            bool      `json:"enabled"`
	BindAddress        string    `json:"bind_address"`
	Port               int       `json:"port"`
	Username           string    `json:"username"`
	PasswordConfigured bool      `json:"password_configured"`
	AllowLocalhostOnly bool      `json:"allow_localhost_only"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ProfileUpdate struct {
	Enabled            *bool
	BindAddress        string
	Port               *int
	Username           string
	Password           string
	AllowLocalhostOnly *bool
}

type ServiceStatus struct {
	Enabled bool         `json:"enabled"`
	Running bool         `json:"running"`
	Address string       `json:"address"`
	State   ServiceState `json:"state"`
}

func DefaultProfile() ManagedWebDAVProfile {
	return ManagedWebDAVProfile{
		ID:                 "managed-localhost-webdav",
		Name:               "Managed Localhost WebDAV",
		BindAddress:        "127.0.0.1",
		Port:               19080,
		Username:           "media",
		AllowLocalhostOnly: true,
	}
}

func (p ManagedWebDAVProfile) View() ProfileView {
	return ProfileView{
		ID:                 p.ID,
		Name:               p.Name,
		Enabled:            p.Enabled,
		BindAddress:        p.BindAddress,
		Port:               p.Port,
		Username:           p.Username,
		PasswordConfigured: p.PasswordHash != "",
		AllowLocalhostOnly: p.AllowLocalhostOnly,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}
