package auth115

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Service struct {
	mu          sync.Mutex
	provider    AuthProvider
	provisioner StorageProvisioner
	config      Config
	now         func() time.Time
	sessions    map[string]*session
}

func NewService(provider AuthProvider, provisioner StorageProvisioner, config Config) *Service {
	return &Service{
		provider:    provider,
		provisioner: provisioner,
		config:      config.normalized(),
		now:         time.Now,
		sessions:    make(map[string]*session),
	}
}

func (s *Service) Start(ctx context.Context) (*StartResponse, error) {
	clientID := s.config.ClientID()
	if clientID == "" {
		return nil, authError(CodeConfigRequired, nil)
	}
	if s.provider == nil {
		return nil, authError(CodeProviderError, fmt.Errorf("provider is not configured"))
	}

	id, err := randomURLString(32)
	if err != nil {
		return nil, authError(CodeProviderError, err)
	}
	verifier, err := randomURLString(32)
	if err != nil {
		return nil, authError(CodeProviderError, err)
	}

	now := s.now()
	item := &session{
		id:        id,
		verifier:  verifier,
		expiresAt: now.Add(s.config.SessionTTL),
		state:     StateCreated,
		done:      make(chan struct{}),
	}
	s.mu.Lock()
	s.pruneLocked(now)
	if s.activeCountLocked() >= s.config.MaxActiveSessions {
		s.mu.Unlock()
		return nil, authError(CodeTooManySessions, nil)
	}
	s.sessions[id] = item
	s.mu.Unlock()

	device, err := s.provider.StartDeviceCode(ctx, clientID, verifier)
	if err != nil {
		s.fail(id, authError(CodeProviderError, err))
		return nil, authError(CodeProviderError, err)
	}
	if device == nil || device.UID == "" {
		err = fmt.Errorf("provider returned an invalid device code")
		s.fail(id, authError(CodeProviderError, err))
		return nil, authError(CodeProviderError, err)
	}

	s.mu.Lock()
	if current := s.sessions[id]; current != nil {
		if s.now().After(current.expiresAt) {
			current.state = StateExpired
			s.closeLocked(current)
			s.mu.Unlock()
			return nil, authError(CodeSessionExpired, nil)
		}
		current.uid = device.UID
		current.timestamp = device.Time
		current.sign = device.Sign
		current.qrCode = device.QRCode
		current.state = StateWaiting
	}
	s.mu.Unlock()

	return &StartResponse{
		SessionID: id,
		State:     StateWaiting,
		QRCode:    device.QRCode,
		ExpiresAt: item.expiresAt,
	}, nil
}

func (s *Service) Status(ctx context.Context, id string) (*StatusResponse, error) {
	if id == "" {
		return nil, authError(CodeInvalidArgument, nil)
	}
	s.mu.Lock()
	item, err := s.getSessionLocked(id)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if item.state == StateWaiting {
		uid, timestamp, sign := item.uid, item.timestamp, item.sign
		s.mu.Unlock()
		status, providerErr := s.provider.GetQRStatus(ctx, uid, timestamp, sign)
		if providerErr != nil {
			return nil, authError(CodeProviderError, providerErr)
		}
		if status == nil {
			return nil, authError(CodeProviderError, fmt.Errorf("provider returned an invalid QR status"))
		}
		s.mu.Lock()
		if current := s.sessions[id]; current != nil {
			current.provider = *status
		}
		item = s.sessions[id]
	}
	response := s.statusResponseLocked(item)
	s.mu.Unlock()
	return response, nil
}

func (s *Service) Complete(ctx context.Context, id string) (*StorageResult, error) {
	if id == "" {
		return nil, authError(CodeInvalidArgument, nil)
	}
	for {
		s.mu.Lock()
		item, err := s.getSessionLocked(id)
		if err != nil {
			s.mu.Unlock()
			return nil, err
		}
		switch item.state {
		case StateReady:
			result := cloneStorageResult(item.storage)
			s.mu.Unlock()
			return result, nil
		case StateCanceled:
			s.mu.Unlock()
			return nil, authError(CodeStateConflict, nil)
		case StateExpired:
			s.mu.Unlock()
			return nil, authError(CodeSessionExpired, nil)
		case StateFailed:
			failure := item.lastError
			if failure == nil {
				failure = authError(CodeExchangeFailed, nil)
			}
			s.mu.Unlock()
			return nil, failure
		case StateExchanging, StatePersisting:
			done := item.done
			s.mu.Unlock()
			select {
			case <-done:
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		case StateWaiting:
			item.state = StateExchanging
			uid, verifier := item.uid, item.verifier
			s.mu.Unlock()
			return s.exchangeAndPersist(ctx, id, uid, verifier)
		default:
			s.mu.Unlock()
			return nil, authError(CodeStateConflict, nil)
		}
	}
}

func (s *Service) exchangeAndPersist(ctx context.Context, id, uid, verifier string) (*StorageResult, error) {
	result, err := s.provider.ExchangeCode(ctx, uid, verifier)
	if err != nil || result == nil || result.AccessToken == "" || result.RefreshToken == "" {
		if err == nil {
			err = fmt.Errorf("provider returned an invalid token pair")
		}
		failure := authError(CodeExchangeFailed, err)
		s.fail(id, failure)
		return nil, failure
	}

	s.mu.Lock()
	item := s.sessions[id]
	if item == nil || item.state != StateExchanging {
		s.mu.Unlock()
		return nil, authError(CodeStateConflict, nil)
	}
	item.state = StatePersisting
	s.mu.Unlock()

	if s.provisioner == nil {
		err = fmt.Errorf("storage provisioner is not configured")
	} else {
		var storage StorageResult
		storage, err = s.provisioner.Provision(ctx, *result)
		if err == nil {
			s.ready(id, storage)
			return &storage, nil
		}
	}
	failure := authError(CodePersistenceFailed, err)
	s.fail(id, failure)
	return nil, failure
}

func (s *Service) Cancel(id string) error {
	if id == "" {
		return authError(CodeInvalidArgument, nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.getSessionLocked(id)
	if err != nil {
		return err
	}
	switch item.state {
	case StateCanceled:
		return nil
	case StateExchanging, StatePersisting, StateReady:
		return authError(CodeStateConflict, nil)
	case StateExpired:
		return authError(CodeSessionExpired, nil)
	default:
		item.state = StateCanceled
		s.closeLocked(item)
		return nil
	}
}

func (s *Service) Import(ctx context.Context, accessToken, refreshToken string) (*StorageResult, error) {
	if accessToken == "" || refreshToken == "" {
		return nil, authError(CodeInvalidArgument, nil)
	}
	if s.provisioner == nil {
		return nil, authError(CodePersistenceFailed, fmt.Errorf("storage provisioner is not configured"))
	}
	result, err := s.provisioner.Provision(ctx, TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, authError(CodePersistenceFailed, err)
	}
	return &result, nil
}

func (s *Service) RetryPersistence(ctx context.Context) (*StorageResult, error) {
	if s.provisioner == nil {
		return nil, authError(CodePersistenceFailed, fmt.Errorf("storage provisioner is not configured"))
	}
	result, err := s.provisioner.Retry(ctx)
	if err != nil {
		return nil, authError(CodePersistenceFailed, err)
	}
	return &result, nil
}

func (s *Service) Capabilities() CapabilitiesResponse {
	return CapabilitiesResponse{
		PKCEAvailable:        true,
		TokenImportAvailable: true,
		ClientIDConfigured:   s.config.ClientID() != "",
	}
}

func (s *Service) StorageStatus() StorageStatus {
	if s.provisioner == nil {
		return StorageStatus{}
	}
	return s.provisioner.Status()
}

func (s *Service) getSessionLocked(id string) (*session, error) {
	item := s.sessions[id]
	if item == nil {
		return nil, authError(CodeSessionNotFound, nil)
	}
	if s.now().After(item.expiresAt) && !isTerminal(item.state) {
		item.state = StateExpired
		s.closeLocked(item)
	}
	return item, nil
}

func (s *Service) pruneLocked(now time.Time) {
	for _, item := range s.sessions {
		if now.After(item.expiresAt) && !isTerminal(item.state) {
			item.state = StateExpired
			s.closeLocked(item)
		}
	}
}

func (s *Service) activeCountLocked() int {
	count := 0
	for _, item := range s.sessions {
		if !isTerminal(item.state) {
			count++
		}
	}
	return count
}

func (s *Service) fail(id string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item := s.sessions[id]; item != nil {
		item.state = StateFailed
		item.lastError = err
		s.closeLocked(item)
	}
}

func (s *Service) ready(id string, result StorageResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item := s.sessions[id]; item != nil {
		item.state = StateReady
		item.storage = cloneStorageResult(&result)
		s.closeLocked(item)
	}
}

func (s *Service) statusResponseLocked(item *session) *StatusResponse {
	response := &StatusResponse{
		SessionID:    item.id,
		State:        item.state,
		ProviderCode: item.provider.Status,
		Message:      item.provider.Message,
		Version:      item.provider.Version,
		ExpiresAt:    item.expiresAt,
		Storage:      cloneStorageResult(item.storage),
	}
	return response
}

func cloneStorageResult(result *StorageResult) *StorageResult {
	if result == nil {
		return nil
	}
	copy := *result
	return &copy
}
