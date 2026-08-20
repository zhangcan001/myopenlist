package auth115

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_115_open "github.com/OpenListTeam/OpenList/v4/drivers/115_open"
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	log "github.com/sirupsen/logrus"
)

type fakeAuthProvider struct {
	status          QRStatusResult
	startErr        error
	deviceCalls     atomic.Int32
	statusCalls     atomic.Int32
	exchangeCalls   atomic.Int32
	exchangeStarted chan struct{}
	exchangeRelease chan struct{}
	startOnce       sync.Once
}

func (p *fakeAuthProvider) StartDeviceCode(context.Context, string, string) (*DeviceCodeResult, error) {
	p.deviceCalls.Add(1)
	if p.startErr != nil {
		return nil, p.startErr
	}
	return &DeviceCodeResult{UID: "uid-1", Time: 123, QRCode: "qr-data", Sign: "private-sign"}, nil
}

func (p *fakeAuthProvider) GetQRStatus(context.Context, string, int64, string) (*QRStatusResult, error) {
	p.statusCalls.Add(1)
	return &p.status, nil
}

func (p *fakeAuthProvider) ExchangeCode(context.Context, string, string) (*TokenPairResult, error) {
	p.exchangeCalls.Add(1)
	if p.exchangeStarted != nil {
		p.startOnce.Do(func() { close(p.exchangeStarted) })
		<-p.exchangeRelease
	}
	return &TokenPairResult{AccessToken: "access-secret", RefreshToken: "refresh-secret"}, nil
}

type fakeProvisioner struct {
	mu     sync.Mutex
	pairs  []TokenPair
	result StorageResult
}

func (p *fakeProvisioner) Provision(_ context.Context, pair TokenPair) (StorageResult, error) {
	return p.record(pair)
}

func (p *fakeProvisioner) Retry(context.Context) (StorageResult, error) {
	return p.result, nil
}

func (p *fakeProvisioner) Status() StorageStatus {
	return StorageStatus{StorageID: p.result.StorageID, MountPath: p.result.MountPath, Connected: p.result.Connected}
}

func (p *fakeProvisioner) record(pair TokenPair) (StorageResult, error) {
	p.mu.Lock()
	p.pairs = append(p.pairs, pair)
	p.mu.Unlock()
	return p.result, nil
}

func newTestService(provider *fakeAuthProvider, provisioner *fakeProvisioner) *Service {
	service := NewService(provider, provisioner, Config{BuiltinClientID: "test-client"})
	return service
}

func TestStartWithoutClientIDReturnsConfigRequired(t *testing.T) {
	t.Setenv("OPENLIST_115_CLIENT_ID", "")
	provider := &fakeAuthProvider{}
	service := NewService(provider, &fakeProvisioner{}, Config{})

	_, err := service.Start(context.Background())
	assertAuthCode(t, err, CodeConfigRequired)
	if got := provider.deviceCalls.Load(); got != 0 {
		t.Fatalf("provider was called %d times", got)
	}
}

func TestStartCreatesSecureSession(t *testing.T) {
	provider := &fakeAuthProvider{}
	service := newTestService(provider, &fakeProvisioner{})
	first, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.SessionID == second.SessionID {
		t.Fatal("session IDs were reused")
	}
	service.mu.Lock()
	verifier := service.sessions[first.SessionID].verifier
	service.mu.Unlock()
	if len(verifier) < 40 {
		t.Fatalf("verifier is too short: %d", len(verifier))
	}
	encoded, _ := json.Marshal(first)
	if string(encoded) == "" || containsAny(string(encoded), "private-sign", "verifier") {
		t.Fatalf("start response leaked private session data: %s", encoded)
	}
}

func TestStatusPreservesProviderStatusWithoutInventedMapping(t *testing.T) {
	provider := &fakeAuthProvider{status: QRStatusResult{Status: 7, Message: "provider message", Version: "v1"}}
	service := newTestService(provider, &fakeProvisioner{})
	started, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(context.Background(), started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if status.ProviderCode != 7 || status.Message != "provider message" || status.Version != "v1" {
		t.Fatalf("provider status was changed: %+v", status)
	}
}

func TestConcurrentCompleteSingleExchange(t *testing.T) {
	provider := &fakeAuthProvider{exchangeStarted: make(chan struct{}), exchangeRelease: make(chan struct{})}
	provisioner := &fakeProvisioner{result: StorageResult{StorageID: 1, MountPath: "/115", Connected: true, State: StateReady}}
	service := newTestService(provider, provisioner)
	started, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 20)
	for range 20 {
		go func() {
			_, completeErr := service.Complete(context.Background(), started.SessionID)
			results <- completeErr
		}()
	}
	<-provider.exchangeStarted
	close(provider.exchangeRelease)
	for range 20 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if got := provider.exchangeCalls.Load(); got != 1 {
		t.Fatalf("exchange calls = %d, want 1", got)
	}
}

func TestCompleteDoesNotLeakToken(t *testing.T) {
	provider := &fakeAuthProvider{}
	provisioner := &fakeProvisioner{result: StorageResult{StorageID: 2, MountPath: "/115", Connected: true, State: StateReady}}
	service := newTestService(provider, provisioner)
	started, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Complete(context.Background(), started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	if containsAny(string(encoded), "access-secret", "refresh-secret", "access_token", "refresh_token") {
		t.Fatalf("complete response leaked token data: %s", encoded)
	}
}

func TestCompleteIdempotent(t *testing.T) {
	provider := &fakeAuthProvider{}
	provisioner := &fakeProvisioner{result: StorageResult{StorageID: 3, MountPath: "/115", Connected: true, State: StateReady}}
	service := newTestService(provider, provisioner)
	started, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Complete(context.Background(), started.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Complete(context.Background(), started.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := provider.exchangeCalls.Load(); got != 1 {
		t.Fatalf("exchange calls = %d, want 1", got)
	}
}

func TestCanceledSessionCannotComplete(t *testing.T) {
	service := newTestService(&fakeAuthProvider{}, &fakeProvisioner{})
	started, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Cancel(started.SessionID); err != nil {
		t.Fatal(err)
	}
	_, err = service.Complete(context.Background(), started.SessionID)
	assertAuthCode(t, err, CodeStateConflict)
}

func TestSessionExpiry(t *testing.T) {
	clock := time.Now()
	service := newTestService(&fakeAuthProvider{}, &fakeProvisioner{})
	service.now = func() time.Time { return clock }
	started, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(11 * time.Minute)
	_, err = service.Complete(context.Background(), started.SessionID)
	assertAuthCode(t, err, CodeSessionExpired)
}

func TestImportTokenPairCreatesManagedStorage(t *testing.T) {
	provisioner := &fakeProvisioner{result: StorageResult{StorageID: 4, MountPath: "/115", Connected: true, State: StateReady}}
	provider := &fakeAuthProvider{}
	service := newTestService(provider, provisioner)
	result, err := service.Import(context.Background(), "access-secret", "refresh-secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.StorageID != 4 || result.MountPath != "/115" {
		t.Fatalf("unexpected import result: %+v", result)
	}
	provisioner.mu.Lock()
	defer provisioner.mu.Unlock()
	if len(provisioner.pairs) != 1 || provisioner.pairs[0].AccessToken != "access-secret" || provisioner.pairs[0].RefreshToken != "refresh-secret" {
		t.Fatalf("unexpected pair passed to persistence: %+v", provisioner.pairs)
	}
	encoded, _ := json.Marshal(result)
	if containsAny(string(encoded), "access-secret", "refresh-secret") {
		t.Fatalf("import response leaked token data: %s", encoded)
	}
}

func TestAuthLogsNeverContainTokens(t *testing.T) {
	var output bytes.Buffer
	logger := log.StandardLogger()
	previous := logger.Out
	logger.SetOutput(&output)
	t.Cleanup(func() { logger.SetOutput(previous) })

	provider := &fakeAuthProvider{startErr: errors.New("SUPER_SECRET_ACCESS SUPER_SECRET_REFRESH")}
	service := newTestService(provider, &fakeProvisioner{})
	_, err := service.Start(context.Background())
	if err == nil || strings.Contains(err.Error(), "SUPER_SECRET") || strings.Contains(output.String(), "SUPER_SECRET") {
		t.Fatalf("token material reached an error or log: err=%v log=%s", err, output.String())
	}
}

func TestExisting115StoragePreservesConfiguration(t *testing.T) {
	original := _115_open.Addition{
		RootID:         driver.RootID{RootFolderID: "folder-1"},
		OrderBy:        "file_size",
		OrderDirection: "desc",
		LimitRate:      2.5,
		PageSize:       321,
		AccessToken:    "old-access",
		RefreshToken:   "old-refresh",
	}
	bytes, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := mergeTokenPair(string(bytes), TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"})
	if err != nil {
		t.Fatal(err)
	}
	var got _115_open.Addition
	if err = json.Unmarshal([]byte(merged), &got); err != nil {
		t.Fatal(err)
	}
	if got.RootFolderID != original.RootFolderID || got.OrderBy != original.OrderBy || got.OrderDirection != original.OrderDirection || got.LimitRate != original.LimitRate || got.PageSize != original.PageSize {
		t.Fatalf("configuration changed during token merge: %+v", got)
	}
	if got.AccessToken != "new-access" || got.RefreshToken != "new-refresh" {
		t.Fatalf("tokens were not replaced: %+v", got)
	}
}

func assertAuthCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s", want)
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Code != want {
		t.Fatalf("error = %v, want %s", err, want)
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
