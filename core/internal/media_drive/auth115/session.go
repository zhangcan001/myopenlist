package auth115

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"time"
)

type session struct {
	id         string
	verifier   string
	uid        string
	timestamp  int64
	sign       string
	expiresAt  time.Time
	state      SessionState
	qrCode     string
	provider   QRStatusResult
	storage    *StorageResult
	lastError  error
	done       chan struct{}
	doneClosed bool
}

func randomURLString(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := cryptorand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (s *Service) closeLocked(item *session) {
	if !item.doneClosed {
		close(item.done)
		item.doneClosed = true
	}
}

func isTerminal(state SessionState) bool {
	switch state {
	case StateReady, StateCanceled, StateExpired, StateFailed:
		return true
	default:
		return false
	}
}
