package webdav

import (
	"crypto/subtle"
	"strings"
	"time"

	"github.com/ProtonMail/bcrypt"
)

func normalizeProfile(profile ManagedWebDAVProfile) ManagedWebDAVProfile {
	defaults := DefaultProfile()
	if profile.ID == "" {
		profile.ID = defaults.ID
	}
	if profile.Name == "" {
		profile.Name = defaults.Name
	}
	if profile.BindAddress == "" {
		profile.BindAddress = defaults.BindAddress
	}
	if profile.Username == "" {
		profile.Username = defaults.Username
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now()
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = profile.CreatedAt
	}
	return profile
}

func validateProfile(profile ManagedWebDAVProfile) error {
	if profile.BindAddress != "127.0.0.1" || !profile.AllowLocalhostOnly {
		return ErrLocalhostOnlyRequired
	}
	if profile.Port < 0 || profile.Port > 65535 {
		return ErrInvalidPort
	}
	if strings.TrimSpace(profile.Username) == "" {
		return ErrInvalidProfile
	}
	return nil
}

func hashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrInvalidProfile
	}
	return bcrypt.Hash(password)
}

func passwordMatches(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}
	calculated, err := bcrypt.Hash(password, hash)
	return err == nil && subtle.ConstantTimeCompare([]byte(calculated), []byte(hash)) == 1
}
