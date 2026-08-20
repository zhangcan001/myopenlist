package sdk

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func refreshTestPolicy() RefreshPolicy {
	return RefreshPolicy{
		MaxAttempts:       3,
		BaseBackoff:       10 * time.Millisecond,
		MaxBackoff:        15 * time.Millisecond,
		JitterFraction:    0,
		CircuitOpenFor:    10 * time.Second,
		RateLimitFallback: 20 * time.Second,
	}
}

func refreshErrorKind(t *testing.T, err error) RefreshErrorKind {
	t.Helper()
	var refreshErr *RefreshError
	if !errors.As(err, &refreshErr) {
		t.Fatalf("error = %v, want RefreshError", err)
	}
	return refreshErr.Kind
}

func TestRefreshErrorClassification(t *testing.T) {
	var networkErr error = &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "offline"}}
	tests := []struct {
		name string
		err  error
		want RefreshErrorKind
	}{
		{name: "context canceled", err: context.Canceled, want: RefreshErrorContext},
		{name: "deadline", err: context.DeadlineExceeded, want: RefreshErrorContext},
		{name: "network", err: networkErr, want: RefreshErrorNetwork},
		{name: "rate limit status", err: &PassportError{HTTPStatus: http.StatusTooManyRequests}, want: RefreshErrorRateLimit},
		{name: "rate limit code", err: &PassportError{Code: 429}, want: RefreshErrorRateLimit},
		{name: "server status", err: &PassportError{HTTPStatus: http.StatusBadGateway}, want: RefreshErrorServer},
		{name: "server code", err: &PassportError{Code: 503}, want: RefreshErrorServer},
		{name: "auth status", err: &PassportError{HTTPStatus: http.StatusUnauthorized}, want: RefreshErrorAuthRequired},
		{name: "auth exact code", err: &PassportError{Code: 40140116}, want: RefreshErrorAuthRequired},
		{name: "permission status", err: &PassportError{HTTPStatus: http.StatusForbidden}, want: RefreshErrorPermission},
		{name: "permission code", err: &PassportError{Code: 403}, want: RefreshErrorPermission},
		{name: "unknown", err: errors.New("unexpected refresh failure"), want: RefreshErrorUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyRefreshError(nil, test.err); got != test.want {
				t.Fatalf("kind = %s, want %s", got, test.want)
			}
		})
	}
}

func TestRefreshNetworkFailureUsesBoundedBackoff(t *testing.T) {
	var refreshCalls atomic.Int32
	var sleeps []time.Duration
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if refreshCalls.Add(1) < 3 {
			writeTestJSON(w, `{"code":500,"message":"temporary"}`)
			return
		}
		writeTestJSON(w, `{"code":0,"data":{"access_token":"retry-access","refresh_token":"retry-refresh"}}`)
	}))
	client.refreshPolicy = refreshTestPolicy()
	client.sleeper = func(ctx context.Context, delay time.Duration) error {
		sleeps = append(sleeps, delay)
		return nil
	}
	client.randFloat = func() float64 { return 0.5 }

	if _, err := client.RefreshToken(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if got := refreshCalls.Load(); got != 3 {
		t.Fatalf("refresh attempts = %d, want 3", got)
	}
	if !reflect.DeepEqual(sleeps, []time.Duration{10 * time.Millisecond, 15 * time.Millisecond}) {
		t.Fatalf("backoff sleeps = %v", sleeps)
	}
	if status := client.RefreshStatus(); status.State != RefreshCircuitClosed {
		t.Fatalf("status = %+v, want CLOSED", status)
	}
}

func TestRefreshRetryExhaustionOpensCircuit(t *testing.T) {
	var refreshCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalls.Add(1)
		writeTestJSON(w, `{"code":500,"message":"server unavailable"}`)
	}))
	client.refreshPolicy = refreshTestPolicy()
	client.sleeper = func(context.Context, time.Duration) error { return nil }
	client.now = func() time.Time { return time.Unix(100, 0) }

	err := func() error {
		_, err := client.RefreshToken(context.Background())
		return err
	}()
	if got := refreshErrorKind(t, err); got != RefreshErrorServer {
		t.Fatalf("kind = %s, want SERVER", got)
	}
	status := client.RefreshStatus()
	if status.State != RefreshCircuitOpen || status.LastErrorKind != RefreshErrorServer {
		t.Fatalf("status = %+v, want OPEN/SERVER", status)
	}
	if _, err := client.RefreshToken(context.Background()); refreshErrorKind(t, err) != RefreshErrorServer {
		t.Fatalf("open-circuit error = %v, want SERVER", err)
	}
	if got := refreshCalls.Load(); got != 3 {
		t.Fatalf("refresh attempts after open = %d, want 3", got)
	}
}

func TestRefreshBackoffHonorsContextCancellation(t *testing.T) {
	var refreshCalls atomic.Int32
	sleepStarted := make(chan struct{})
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalls.Add(1)
		writeTestJSON(w, `{"code":500,"message":"temporary"}`)
	}))
	client.refreshPolicy = refreshTestPolicy()
	client.sleeper = func(ctx context.Context, delay time.Duration) error {
		close(sleepStarted)
		<-ctx.Done()
		return ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := client.RefreshToken(ctx)
		result <- err
	}()
	waitTest(t, sleepStarted, "refresh backoff")
	cancel()
	err := <-result
	if got := refreshErrorKind(t, err); got != RefreshErrorContext {
		t.Fatalf("kind = %s, want CONTEXT", got)
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh attempts = %d, want 1", got)
	}
	if status := client.RefreshStatus(); status.State != RefreshCircuitClosed || status.LastErrorKind != RefreshErrorContext {
		t.Fatalf("status = %+v, want CLOSED/CONTEXT", status)
	}
}

func TestRefreshRateLimitOpensCircuitWithoutImmediateRetry(t *testing.T) {
	var refreshCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"code":429,"message":"slow down"}`)
	}))
	client.refreshPolicy = refreshTestPolicy()
	client.now = func() time.Time { return time.Unix(100, 0) }

	_, err := client.RefreshToken(context.Background())
	if got := refreshErrorKind(t, err); got != RefreshErrorRateLimit {
		t.Fatalf("kind = %s, want RATE_LIMIT", got)
	}
	status := client.RefreshStatus()
	if status.State != RefreshCircuitOpen || !status.RetryAt.Equal(time.Unix(100, 0).Add(10*time.Second)) {
		t.Fatalf("status = %+v, want OPEN with ten-second retry", status)
	}
	if _, err := client.RefreshToken(context.Background()); refreshErrorKind(t, err) != RefreshErrorRateLimit {
		t.Fatalf("open-circuit error = %v, want RATE_LIMIT", err)
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh attempts = %d, want 1", got)
	}
}

func TestRefreshCircuitHalfOpenAllowsSingleProbe(t *testing.T) {
	var refreshCalls atomic.Int32
	probeStarted := make(chan struct{})
	releaseProbe := make(chan struct{})
	var probeOnce sync.Once
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch refreshCalls.Add(1) {
		case 1:
			writeTestJSON(w, `{"code":500,"message":"temporary"}`)
		case 2:
			probeOnce.Do(func() { close(probeStarted) })
			<-releaseProbe
			writeTestJSON(w, `{"code":0,"data":{"access_token":"probe-access","refresh_token":"probe-refresh"}}`)
		default:
			t.Fatalf("unexpected refresh call")
		}
	}))
	policy := refreshTestPolicy()
	policy.MaxAttempts = 1
	client.refreshPolicy = policy
	var nowNanos atomic.Int64
	base := time.Unix(100, 0)
	nowNanos.Store(base.UnixNano())
	client.now = func() time.Time { return time.Unix(0, nowNanos.Load()) }
	client.sleeper = func(context.Context, time.Duration) error { return nil }

	if _, err := client.RefreshToken(context.Background()); refreshErrorKind(t, err) != RefreshErrorServer {
		t.Fatalf("initial error = %v, want SERVER", err)
	}
	nowNanos.Store(base.Add(11 * time.Second).UnixNano())
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func() {
			_, err := client.RefreshToken(context.Background())
			results <- err
		}()
	}
	waitTest(t, probeStarted, "half-open probe")
	waitRefreshJoiners(t, client, 19)
	close(releaseProbe)
	for i := 0; i < 20; i++ {
		if err := <-results; err != nil {
			t.Fatalf("probe request failed: %v", err)
		}
	}
	if got := refreshCalls.Load(); got != 2 {
		t.Fatalf("refresh calls = %d, want 2", got)
	}
	if status := client.RefreshStatus(); status.State != RefreshCircuitClosed {
		t.Fatalf("status = %+v, want CLOSED", status)
	}
}

func TestRefreshHalfOpenFailureReopensCircuit(t *testing.T) {
	var refreshCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalls.Add(1)
		writeTestJSON(w, `{"code":500,"message":"still unavailable"}`)
	}))
	policy := refreshTestPolicy()
	policy.MaxAttempts = 1
	client.refreshPolicy = policy
	var nowNanos atomic.Int64
	base := time.Unix(100, 0)
	nowNanos.Store(base.UnixNano())
	client.now = func() time.Time { return time.Unix(0, nowNanos.Load()) }

	if _, err := client.RefreshToken(context.Background()); refreshErrorKind(t, err) != RefreshErrorServer {
		t.Fatalf("initial error = %v, want SERVER", err)
	}
	nowNanos.Store(base.Add(11 * time.Second).UnixNano())
	if _, err := client.RefreshToken(context.Background()); refreshErrorKind(t, err) != RefreshErrorServer {
		t.Fatalf("half-open error = %v, want SERVER", err)
	}
	status := client.RefreshStatus()
	if status.State != RefreshCircuitOpen || !status.RetryAt.Equal(base.Add(21*time.Second)) {
		t.Fatalf("status = %+v, want reopened circuit", status)
	}
	if got := refreshCalls.Load(); got != 2 {
		t.Fatalf("refresh calls = %d, want 2", got)
	}
}

func TestInvalidRefreshTokenEntersAuthRequired(t *testing.T) {
	var refreshCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalls.Add(1)
		writeTestJSON(w, `{"code":40140116,"message":"invalid refresh token"}`)
	}))
	client.refreshPolicy.MaxAttempts = 1
	if _, err := client.RefreshToken(context.Background()); refreshErrorKind(t, err) != RefreshErrorAuthRequired {
		t.Fatalf("initial error = %v, want AUTH_REQUIRED", err)
	}
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func() {
			_, err := client.RefreshToken(context.Background())
			results <- err
		}()
	}
	for i := 0; i < 20; i++ {
		if err := <-results; refreshErrorKind(t, err) != RefreshErrorAuthRequired {
			t.Fatalf("fast-fail error = %v, want AUTH_REQUIRED", err)
		}
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls = %d, want 1", got)
	}
	if status := client.RefreshStatus(); status.State != RefreshCircuitAuthRequired {
		t.Fatalf("status = %+v, want AUTH_REQUIRED", status)
	}
}

func TestNewTokenPairResetsAuthRequired(t *testing.T) {
	var refreshCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if refreshCalls.Add(1) == 1 {
			writeTestJSON(w, `{"code":40140116,"message":"invalid refresh token"}`)
			return
		}
		writeTestJSON(w, `{"code":0,"data":{"access_token":"fresh-access","refresh_token":"fresh-refresh"}}`)
	}))
	client.refreshPolicy.MaxAttempts = 1
	if _, err := client.RefreshToken(context.Background()); refreshErrorKind(t, err) != RefreshErrorAuthRequired {
		t.Fatalf("initial error = %v, want AUTH_REQUIRED", err)
	}
	client.setTokenPair("reauthorized-access", "reauthorized-refresh", false)
	if status := client.RefreshStatus(); status.State != RefreshCircuitClosed {
		t.Fatalf("status after token pair = %+v, want CLOSED", status)
	}
	if _, err := client.RefreshToken(context.Background()); err != nil {
		t.Fatalf("refresh after reauthorization failed: %v", err)
	}
	if got := refreshCalls.Load(); got != 2 {
		t.Fatalf("refresh calls = %d, want 2", got)
	}
}

func TestAccessTokenOnlyDoesNotResetAuthRequired(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, `{"code":40140116,"message":"invalid refresh token"}`)
	}))
	client.refreshPolicy.MaxAttempts = 1
	if _, err := client.RefreshToken(context.Background()); refreshErrorKind(t, err) != RefreshErrorAuthRequired {
		t.Fatalf("initial error = %v, want AUTH_REQUIRED", err)
	}
	client.SetAccessToken("new-access-only")
	if status := client.RefreshStatus(); status.State != RefreshCircuitAuthRequired {
		t.Fatalf("status after access-only update = %+v, want AUTH_REQUIRED", status)
	}
}

func TestRefreshRetrySuccessCallbackExactlyOnce(t *testing.T) {
	var refreshCalls atomic.Int32
	var callbackCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if refreshCalls.Add(1) == 1 {
			writeTestJSON(w, `{"code":500,"message":"temporary"}`)
			return
		}
		writeTestJSON(w, `{"code":0,"data":{"access_token":"callback-access","refresh_token":"callback-refresh"}}`)
	}))
	client.refreshPolicy = refreshTestPolicy()
	client.sleeper = func(context.Context, time.Duration) error { return nil }
	client.SetOnRefreshToken(func(string, string) { callbackCalls.Add(1) })
	if _, err := client.RefreshToken(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if got := callbackCalls.Load(); got != 1 {
		t.Fatalf("callback calls = %d, want 1", got)
	}
}

func TestRefreshStatusDoesNotExposeTokens(t *testing.T) {
	typeOfStatus := reflect.TypeOf(RefreshStatus{})
	for i := 0; i < typeOfStatus.NumField(); i++ {
		name := typeOfStatus.Field(i).Name
		if name == "AccessToken" || name == "RefreshToken" || name == "TokenGeneration" {
			t.Fatalf("RefreshStatus exposes token field %q", name)
		}
	}
	client := New(WithAccessToken("secret-access"), WithRefreshToken("secret-refresh"))
	status := client.RefreshStatus()
	if status.State != RefreshCircuitClosed || status.LastErrorKind != "" || !status.RetryAt.IsZero() {
		t.Fatalf("initial status = %+v", status)
	}
}

func TestExplicitRefreshAfterSupersededFlightUsesCurrentToken(t *testing.T) {
	var refreshCalls atomic.Int32
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var firstOnce sync.Once
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := refreshCalls.Add(1)
		if call == 1 {
			if got := r.FormValue("refresh_token"); got != testOldRefresh {
				t.Errorf("first refresh token = %q, want old token", got)
			}
			firstOnce.Do(func() { close(firstStarted) })
			<-releaseFirst
			writeTestJSON(w, `{"code":0,"data":{"access_token":"stale-access","refresh_token":"stale-refresh"}}`)
			return
		}
		close(secondStarted)
		if got := r.FormValue("refresh_token"); got != "reauth-refresh" {
			t.Errorf("second refresh token = %q, want reauth token", got)
		}
		writeTestJSON(w, `{"code":0,"data":{"access_token":"final-access","refresh_token":"final-refresh"}}`)
	}))
	before := client.snapshotToken().generation
	leader := make(chan error, 1)
	go func() {
		_, err := client.RefreshToken(context.Background())
		leader <- err
	}()
	waitTest(t, firstStarted, "first refresh")
	client.setTokenPair("reauth-access", "reauth-refresh", false)
	waiterReady := make(chan struct{})
	waiter := make(chan error, 1)
	go func() {
		close(waiterReady)
		_, err := client.RefreshToken(context.Background())
		waiter <- err
	}()
	waitTest(t, waiterReady, "explicit waiter")
	select {
	case <-secondStarted:
		t.Fatal("second refresh started before stale flight completed")
	default:
	}
	close(releaseFirst)
	if err := <-leader; !errors.Is(err, ErrRefreshSuperseded) {
		t.Fatalf("leader error = %v, want ErrRefreshSuperseded", err)
	}
	waitTest(t, secondStarted, "current-token refresh")
	if err := <-waiter; err != nil {
		t.Fatalf("current-token refresh failed: %v", err)
	}
	snapshot := client.snapshotToken()
	if snapshot.accessToken != "final-access" || snapshot.refreshToken != "final-refresh" {
		t.Fatalf("final token pair = %q/%q", snapshot.accessToken, snapshot.refreshToken)
	}
	if snapshot.generation != before+2 {
		t.Fatalf("generation = %d, want %d", snapshot.generation, before+2)
	}
	if got := refreshCalls.Load(); got != 2 {
		t.Fatalf("refresh calls = %d, want 2", got)
	}
}
