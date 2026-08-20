package auth115

import (
	"context"
	"os"
	"strings"
	"time"
)

type SessionState string

const (
	StateCreated        SessionState = "CREATED"
	StateWaiting        SessionState = "WAITING"
	StateExchanging     SessionState = "EXCHANGING"
	StatePersisting     SessionState = "PERSISTING"
	StateReady          SessionState = "READY"
	StateCanceled       SessionState = "CANCELED"
	StateExpired        SessionState = "EXPIRED"
	StateFailed         SessionState = "FAILED"
	StateConfigRequired SessionState = "CONFIG_REQUIRED"
)

type ErrorCode string

const (
	CodeConfigRequired    ErrorCode = "CONFIG_REQUIRED"
	CodeProviderError     ErrorCode = "PROVIDER_ERROR"
	CodeSessionNotFound   ErrorCode = "SESSION_NOT_FOUND"
	CodeSessionExpired    ErrorCode = "SESSION_EXPIRED"
	CodeSessionCanceled   ErrorCode = "SESSION_CANCELED"
	CodeExchangeFailed    ErrorCode = "EXCHANGE_FAILED"
	CodePersistenceFailed ErrorCode = "PERSISTENCE_FAILED"
	CodeStorageConflict   ErrorCode = "STORAGE_CONFLICT"
	CodeStorageInitFailed ErrorCode = "STORAGE_INIT_FAILED"
	CodeStorageNotFound   ErrorCode = "STORAGE_NOT_FOUND"
	CodeTooManySessions   ErrorCode = "TOO_MANY_SESSIONS"
	CodeStateConflict     ErrorCode = "STATE_CONFLICT"
	CodeInvalidArgument   ErrorCode = "INVALID_ARGUMENT"
)

// AuthError intentionally exposes only a stable code to callers. Provider and
// storage errors are retained for internal errors.Is/errors.As use, but are
// never returned as API messages.
type AuthError struct {
	Code  ErrorCode
	Cause error
}

func (e *AuthError) Error() string { return string(e.Code) }

func (e *AuthError) Unwrap() error { return e.Cause }

func authError(code ErrorCode, cause error) error {
	return &AuthError{Code: code, Cause: cause}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type DeviceCodeResult struct {
	UID    string
	Time   int64
	QRCode string
	Sign   string
}

type QRStatusResult struct {
	Status  int
	Message string
	Version string
}

type TokenPairResult = TokenPair

type AuthProvider interface {
	StartDeviceCode(ctx context.Context, clientID, codeVerifier string) (*DeviceCodeResult, error)
	GetQRStatus(ctx context.Context, uid string, timestamp int64, sign string) (*QRStatusResult, error)
	ExchangeCode(ctx context.Context, uid, codeVerifier string) (*TokenPairResult, error)
}

type StorageResult struct {
	StorageID uint         `json:"storage_id"`
	MountPath string       `json:"mount_path"`
	Connected bool         `json:"connected"`
	State     SessionState `json:"state"`
}

type TokenPersistenceStatus struct {
	State     string `json:"state"`
	Attempts  int    `json:"attempts"`
	LastError string `json:"last_error,omitempty"`
}

type StorageStatus struct {
	StorageID   uint                   `json:"storage_id"`
	MountPath   string                 `json:"mount_path"`
	Connected   bool                   `json:"connected"`
	Persistence TokenPersistenceStatus `json:"persistence"`
}

type StorageProvisioner interface {
	Provision(ctx context.Context, pair TokenPair) (StorageResult, error)
	Retry(ctx context.Context) (StorageResult, error)
	Status() StorageStatus
}

type Config struct {
	BuiltinClientID   string
	SessionTTL        time.Duration
	MaxActiveSessions int
}

// BuiltinClientID is deliberately empty in the open-source build. A release
// or deployment may provide an approved public client ID through configuration.
var BuiltinClientID string

func DefaultConfig() Config {
	return Config{
		BuiltinClientID:   BuiltinClientID,
		SessionTTL:        10 * time.Minute,
		MaxActiveSessions: 4,
	}
}

func (c Config) normalized() Config {
	if c.SessionTTL <= 0 {
		c.SessionTTL = 10 * time.Minute
	}
	if c.MaxActiveSessions <= 0 {
		c.MaxActiveSessions = 4
	}
	return c
}

func (c Config) ClientID() string {
	if value := strings.TrimSpace(os.Getenv("OPENLIST_115_CLIENT_ID")); value != "" {
		return value
	}
	return strings.TrimSpace(c.BuiltinClientID)
}

type StartResponse struct {
	SessionID string       `json:"session_id"`
	State     SessionState `json:"state"`
	QRCode    string       `json:"qr_code,omitempty"`
	ExpiresAt time.Time    `json:"expires_at"`
}

type StatusResponse struct {
	SessionID    string         `json:"session_id"`
	State        SessionState   `json:"state"`
	ProviderCode int            `json:"provider_status,omitempty"`
	Message      string         `json:"provider_message,omitempty"`
	Version      string         `json:"version,omitempty"`
	ExpiresAt    time.Time      `json:"expires_at"`
	Storage      *StorageResult `json:"storage,omitempty"`
}

type CapabilitiesResponse struct {
	PKCEAvailable        bool `json:"pkce_available"`
	TokenImportAvailable bool `json:"token_import_available"`
	ClientIDConfigured   bool `json:"client_configured"`
}
