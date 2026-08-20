package sdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testOldAccess  = "old-access"
	testOldRefresh = "old-refresh"
	testNewAccess  = "new-access"
	testNewRefresh = "new-refresh"
)

type rewriteTransport struct {
	base *url.URL
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = t.base.Scheme
	clone.URL.Host = t.base.Host
	return http.DefaultTransport.RoundTrip(clone)
}

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return New(WithAccessToken(testOldAccess), WithRefreshToken(testOldRefresh)).SetHttpClient(&http.Client{
		Transport: rewriteTransport{base: base},
	})
}

func waitTest(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitRefreshJoiners(t *testing.T, client *Client, count int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		client.tokenMu.Lock()
		joined := 0
		if client.refresh != nil {
			joined = client.refresh.joined
		}
		client.tokenMu.Unlock()
		if joined == count {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("refresh joiners = %d, want %d", joined, count)
		}
		runtime.Gosched()
	}
}

func writeTestJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, body)
}

func runAuthRequests(client *Client, count int) []error {
	results := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.AuthRequest(context.Background(), ApiUserInfo, http.MethodGet, nil)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	collected := make([]error, 0, count)
	for err := range results {
		collected = append(collected, err)
	}
	return collected
}

func TestConcurrent401SingleRefresh(t *testing.T) {
	const requestCount = 20
	var refreshCalls atomic.Int32
	var oldRequests atomic.Int32
	var unexpectedTokens atomic.Int32
	var callbackCalls atomic.Int32
	var callbackAccess, callbackRefresh string
	allInitial401 := make(chan struct{})
	releaseRefresh := make(chan struct{})
	var allInitialOnce sync.Once

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open/refreshToken":
			refreshCalls.Add(1)
			<-releaseRefresh
			writeTestJSON(w, `{"code":0,"data":{"access_token":"new-access","refresh_token":"new-refresh"}}`)
		case "/open/user/info":
			switch r.Header.Get("Authorization") {
			case "Bearer old-access":
				if oldRequests.Add(1) == requestCount {
					allInitialOnce.Do(func() { close(allInitial401) })
				}
				writeTestJSON(w, `{"state":false,"code":40101,"message":"expired"}`)
			case "Bearer new-access":
				writeTestJSON(w, `{"state":true,"data":{}}`)
			default:
				unexpectedTokens.Add(1)
				writeTestJSON(w, `{"state":false,"code":500,"message":"unexpected token"}`)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	client.SetOnRefreshToken(func(access, refresh string) {
		callbackAccess, callbackRefresh = access, refresh
		callbackCalls.Add(1)
	})
	before := client.snapshotToken().generation

	results := make(chan []error, 1)
	go func() { results <- runAuthRequests(client, requestCount) }()
	waitTest(t, allInitial401, "all initial 401 responses")
	waitRefreshJoiners(t, client, requestCount-1)
	close(releaseRefresh)
	requestErrors := <-results

	for _, err := range requestErrors {
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls = %d, want 1", got)
	}
	if got := callbackCalls.Load(); got != 1 {
		t.Fatalf("callback calls = %d, want 1", got)
	}
	if callbackAccess != testNewAccess || callbackRefresh != testNewRefresh {
		t.Fatalf("callback pair = %q/%q, want %q/%q", callbackAccess, callbackRefresh, testNewAccess, testNewRefresh)
	}
	if got := unexpectedTokens.Load(); got != 0 {
		t.Fatalf("unexpected token requests = %d", got)
	}
	if got := client.snapshotToken().generation; got != before+1 {
		t.Fatalf("generation = %d, want %d", got, before+1)
	}
}

func TestStale401DoesNotRefreshAgain(t *testing.T) {
	var refreshCalls atomic.Int32
	var oldRequests atomic.Int32
	a401 := make(chan struct{})
	refreshStarted := make(chan struct{})
	bStarted := make(chan struct{})
	refreshCommitted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	releaseB401 := make(chan struct{})
	var a401Once, refreshOnce, bOnce, committedOnce sync.Once

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open/refreshToken":
			refreshCalls.Add(1)
			refreshOnce.Do(func() { close(refreshStarted) })
			<-releaseRefresh
			writeTestJSON(w, `{"code":0,"data":{"access_token":"new-access","refresh_token":"new-refresh"}}`)
		case "/open/user/info":
			if r.Header.Get("Authorization") == "Bearer old-access" {
				switch oldRequests.Add(1) {
				case 1:
					a401Once.Do(func() { close(a401) })
				case 2:
					bOnce.Do(func() { close(bStarted) })
					<-releaseB401
				}
				writeTestJSON(w, `{"state":false,"code":40101,"message":"expired"}`)
				return
			}
			writeTestJSON(w, `{"state":true,"data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	client.SetOnRefreshToken(func(string, string) {
		committedOnce.Do(func() { close(refreshCommitted) })
	})
	before := client.snapshotToken().generation

	aResult := make(chan error, 1)
	go func() {
		_, err := client.AuthRequest(context.Background(), ApiUserInfo, http.MethodGet, nil)
		aResult <- err
	}()
	waitTest(t, a401, "request A 401")
	waitTest(t, refreshStarted, "refresh flight")

	bResult := make(chan error, 1)
	go func() {
		_, err := client.AuthRequest(context.Background(), ApiUserInfo, http.MethodGet, nil)
		bResult <- err
	}()
	waitTest(t, bStarted, "request B in-flight old-token request")

	close(releaseRefresh)
	waitTest(t, refreshCommitted, "new token pair commit")
	close(releaseB401)

	if err := <-aResult; err != nil {
		t.Fatalf("request A failed: %v", err)
	}
	if err := <-bResult; err != nil {
		t.Fatalf("request B failed: %v", err)
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls = %d, want 1", got)
	}
	if got := client.snapshotToken().generation; got != before+1 {
		t.Fatalf("generation = %d, want %d", got, before+1)
	}
}

func TestTokenPairNeverObservedMixed(t *testing.T) {
	client := New()
	pairs := [][2]string{{"access-1", "refresh-1"}, {"access-2", "refresh-2"}, {"access-3", "refresh-3"}}
	client.setTokenPair(pairs[0][0], pairs[0][1], false)
	const readerCount = 8
	const iterations = 20000
	start := make(chan struct{})
	bad := make(chan tokenSnapshot, 1)
	var badOnce sync.Once
	var wg sync.WaitGroup

	for i := 0; i < readerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				snapshot := client.snapshotToken()
				valid := false
				for _, pair := range pairs {
					if snapshot.accessToken == pair[0] && snapshot.refreshToken == pair[1] {
						valid = true
						break
					}
				}
				if !valid {
					badOnce.Do(func() { bad <- snapshot })
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			pair := pairs[i%len(pairs)]
			client.setTokenPair(pair[0], pair[1], false)
		}
	}()
	close(start)
	wg.Wait()

	select {
	case snapshot := <-bad:
		t.Fatalf("mixed token pair observed: %q/%q", snapshot.accessToken, snapshot.refreshToken)
	default:
	}
}

func TestConcurrentRefreshFailureShared(t *testing.T) {
	const requestCount = 20
	var refreshCalls atomic.Int32
	var oldRequests atomic.Int32
	allInitial401 := make(chan struct{})
	refreshStarted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	var allInitialOnce, refreshOnce sync.Once

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open/refreshToken":
			refreshCalls.Add(1)
			refreshOnce.Do(func() { close(refreshStarted) })
			<-releaseRefresh
			writeTestJSON(w, `{"code":401,"message":"invalid refresh token"}`)
		case "/open/user/info":
			oldRequests.Add(1)
			if oldRequests.Load() == requestCount {
				allInitialOnce.Do(func() { close(allInitial401) })
			}
			writeTestJSON(w, `{"state":false,"code":40101,"message":"expired"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	before := client.snapshotToken().generation

	results := make(chan []error, 1)
	go func() { results <- runAuthRequests(client, requestCount) }()
	waitTest(t, allInitial401, "all failure-cohort 401 responses")
	waitTest(t, refreshStarted, "failure refresh flight")
	waitRefreshJoiners(t, client, requestCount-1)
	close(releaseRefresh)
	requestErrors := <-results

	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls = %d, want 1", got)
	}
	var first error
	for _, err := range requestErrors {
		if err == nil {
			t.Fatal("request unexpectedly succeeded")
		}
		if first == nil {
			first = err
		} else if err != first {
			t.Fatalf("waiter error pointer differs: %p != %p", err, first)
		}
	}
	if got := client.snapshotToken().generation; got != before {
		t.Fatalf("generation = %d after failure, want %d", got, before)
	}
}

func TestRefreshFailureDoesNotStickForever(t *testing.T) {
	const requestCount = 20
	var refreshCalls atomic.Int32
	var oldRequests atomic.Int32
	allInitial401 := make(chan struct{})
	releaseFirstRefresh := make(chan struct{})
	var allInitialOnce sync.Once

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open/refreshToken":
			switch refreshCalls.Add(1) {
			case 1:
				<-releaseFirstRefresh
				writeTestJSON(w, `{"code":500,"message":"temporary server error"}`)
			case 2:
				writeTestJSON(w, `{"code":0,"data":{"access_token":"new-access","refresh_token":"new-refresh"}}`)
			}
		case "/open/user/info":
			if r.Header.Get("Authorization") == "Bearer old-access" {
				if oldRequests.Add(1) == requestCount {
					allInitialOnce.Do(func() { close(allInitial401) })
				}
				writeTestJSON(w, `{"state":false,"code":40101,"message":"expired"}`)
				return
			}
			writeTestJSON(w, `{"state":true,"data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))

	firstResults := make(chan []error, 1)
	go func() { firstResults <- runAuthRequests(client, requestCount) }()
	waitTest(t, allInitial401, "first failure cohort")
	waitRefreshJoiners(t, client, requestCount-1)
	close(releaseFirstRefresh)
	for _, err := range <-firstResults {
		if err == nil || !strings.Contains(err.Error(), "temporary server error") {
			t.Fatalf("unexpected first-round error: %v", err)
		}
	}

	if _, err := client.AuthRequest(context.Background(), ApiUserInfo, http.MethodGet, nil); err != nil {
		t.Fatalf("second request failed after retryable refresh failure: %v", err)
	}
	if got := refreshCalls.Load(); got != 2 {
		t.Fatalf("refresh calls = %d, want 2", got)
	}
}

func TestRefreshCallbackExactlyOnce(t *testing.T) {
	const requestCount = 20
	var refreshCalls atomic.Int32
	var oldRequests atomic.Int32
	var callbackCalls atomic.Int32
	allInitial401 := make(chan struct{})
	releaseRefresh := make(chan struct{})
	var allInitialOnce sync.Once

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open/refreshToken":
			refreshCalls.Add(1)
			<-releaseRefresh
			writeTestJSON(w, `{"code":0,"data":{"access_token":"new-access","refresh_token":"new-refresh"}}`)
		case "/open/user/info":
			if r.Header.Get("Authorization") == "Bearer old-access" {
				if oldRequests.Add(1) == requestCount {
					allInitialOnce.Do(func() { close(allInitial401) })
				}
				writeTestJSON(w, `{"state":false,"code":40101,"message":"expired"}`)
				return
			}
			writeTestJSON(w, `{"state":true,"data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	client.SetOnRefreshToken(func(access, refresh string) {
		if access != testNewAccess || refresh != testNewRefresh {
			t.Errorf("callback pair = %q/%q", access, refresh)
		}
		callbackCalls.Add(1)
	})

	results := make(chan []error, 1)
	go func() { results <- runAuthRequests(client, requestCount) }()
	waitTest(t, allInitial401, "callback test 401 cohort")
	waitRefreshJoiners(t, client, requestCount-1)
	close(releaseRefresh)
	for _, err := range <-results {
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls = %d, want 1", got)
	}
	if got := callbackCalls.Load(); got != 1 {
		t.Fatalf("callback calls = %d, want 1", got)
	}
}

func TestRefreshWaiterCanCancel(t *testing.T) {
	var refreshCalls atomic.Int32
	var oldRequests atomic.Int32
	leader401 := make(chan struct{})
	waiter401 := make(chan struct{})
	refreshStarted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	var leaderOnce, waiterOnce, refreshOnce sync.Once

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open/refreshToken":
			refreshCalls.Add(1)
			refreshOnce.Do(func() { close(refreshStarted) })
			<-releaseRefresh
			writeTestJSON(w, `{"code":0,"data":{"access_token":"new-access","refresh_token":"new-refresh"}}`)
		case "/open/user/info":
			if r.Header.Get("Authorization") == "Bearer old-access" {
				switch oldRequests.Add(1) {
				case 1:
					leaderOnce.Do(func() { close(leader401) })
				case 2:
					waiterOnce.Do(func() { close(waiter401) })
				}
				writeTestJSON(w, `{"state":false,"code":40101,"message":"expired"}`)
				return
			}
			writeTestJSON(w, `{"state":true,"data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	var callbackCalls atomic.Int32
	client.SetOnRefreshToken(func(string, string) { callbackCalls.Add(1) })

	leaderResult := make(chan error, 1)
	go func() {
		_, err := client.AuthRequest(context.Background(), ApiUserInfo, http.MethodGet, nil)
		leaderResult <- err
	}()
	waitTest(t, leader401, "leader 401")
	waitTest(t, refreshStarted, "leader refresh")

	waiterCtx, cancel := context.WithCancel(context.Background())
	waiterResult := make(chan error, 1)
	go func() {
		_, err := client.AuthRequest(waiterCtx, ApiUserInfo, http.MethodGet, nil)
		waiterResult <- err
	}()
	waitTest(t, waiter401, "waiter 401")
	waitRefreshJoiners(t, client, 1)
	cancel()

	select {
	case err := <-waiterResult:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waiter error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("canceled waiter did not return")
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls before leader release = %d, want 1", got)
	}
	close(releaseRefresh)
	if err := <-leaderResult; err != nil {
		t.Fatalf("leader failed: %v", err)
	}
	if got := callbackCalls.Load(); got != 1 {
		t.Fatalf("callback calls = %d, want 1", got)
	}
}

func TestNormalRequestDoesNotRefresh(t *testing.T) {
	var refreshCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/open/refreshToken" {
			refreshCalls.Add(1)
		}
		writeTestJSON(w, `{"state":true,"data":{}}`)
	}))
	before := client.snapshotToken().generation
	if _, err := client.AuthRequest(context.Background(), ApiUserInfo, http.MethodGet, nil); err != nil {
		t.Fatalf("normal request failed: %v", err)
	}
	if got := refreshCalls.Load(); got != 0 {
		t.Fatalf("refresh calls = %d, want 0", got)
	}
	if got := client.snapshotToken().generation; got != before {
		t.Fatalf("generation = %d, want %d", got, before)
	}
}

func TestNonAuthErrorDoesNotRefresh(t *testing.T) {
	var refreshCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/open/refreshToken" {
			refreshCalls.Add(1)
		}
		writeTestJSON(w, `{"state":false,"code":400,"message":"bad request"}`)
	}))
	before := client.snapshotToken().generation
	_, err := client.AuthRequest(context.Background(), ApiUserInfo, http.MethodGet, nil)
	var sdkErr *Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != 400 {
		t.Fatalf("error = %v, want *Error code 400", err)
	}
	if got := refreshCalls.Load(); got != 0 {
		t.Fatalf("refresh calls = %d, want 0", got)
	}
	if got := client.snapshotToken().generation; got != before {
		t.Fatalf("generation = %d, want %d", got, before)
	}
}

func TestConcurrentExplicitRefreshSingleFlight(t *testing.T) {
	const requestCount = 20
	var refreshCalls atomic.Int32
	var callbackCalls atomic.Int32
	refreshStarted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	var refreshOnce sync.Once
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open/refreshToken" {
			http.NotFound(w, r)
			return
		}
		refreshCalls.Add(1)
		refreshOnce.Do(func() { close(refreshStarted) })
		<-releaseRefresh
		writeTestJSON(w, `{"code":0,"data":{"access_token":"new-access","refresh_token":"new-refresh"}}`)
	}))
	client.SetOnRefreshToken(func(string, string) { callbackCalls.Add(1) })
	before := client.snapshotToken().generation
	start := make(chan struct{})
	allEntered := make(chan struct{})
	var entered atomic.Int32
	var enteredOnce sync.Once
	results := make(chan error, requestCount)
	var wg sync.WaitGroup
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if entered.Add(1) == requestCount {
				enteredOnce.Do(func() { close(allEntered) })
			}
			_, err := client.RefreshToken(context.Background())
			results <- err
		}()
	}
	close(start)
	waitTest(t, allEntered, "all explicit refresh callers")
	waitTest(t, refreshStarted, "explicit refresh flight")
	waitRefreshJoiners(t, client, requestCount-1)
	close(releaseRefresh)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("explicit refresh failed: %v", err)
		}
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls = %d, want 1", got)
	}
	if got := callbackCalls.Load(); got != 1 {
		t.Fatalf("callback calls = %d, want 1", got)
	}
	if got := client.snapshotToken().generation; got != before+1 {
		t.Fatalf("generation = %d, want %d", got, before+1)
	}
}

func TestTokenGenerationIncrementsOncePerRefresh(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, `{"code":0,"data":{"access_token":"new-access","refresh_token":"new-refresh"}}`)
	}))
	before := client.snapshotToken().generation
	if _, err := client.RefreshToken(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if got := client.snapshotToken().generation; got != before+1 {
		t.Fatalf("generation = %d, want %d", got, before+1)
	}
}

func TestCodeToTokenCommitsAtomicPair(t *testing.T) {
	var callbackCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open/deviceCodeToToken" {
			http.NotFound(w, r)
			return
		}
		writeTestJSON(w, `{"code":0,"data":{"access_token":"new-access","refresh_token":"new-refresh"}}`)
	}))
	client.SetOnRefreshToken(func(string, string) { callbackCalls.Add(1) })
	before := client.snapshotToken().generation
	if _, err := client.CodeToToken(context.Background(), "uid", "verifier"); err != nil {
		t.Fatalf("code-to-token failed: %v", err)
	}
	after := client.snapshotToken()
	if after.accessToken != testNewAccess || after.refreshToken != testNewRefresh {
		t.Fatalf("token pair = %q/%q, want %q/%q", after.accessToken, after.refreshToken, testNewAccess, testNewRefresh)
	}
	if after.generation != before+1 {
		t.Fatalf("generation = %d, want %d", after.generation, before+1)
	}
	if got := callbackCalls.Load(); got != 0 {
		t.Fatalf("callback calls = %d, want 0", got)
	}
}

func TestPublicSettersAreThreadSafe(t *testing.T) {
	client := New()
	client.SetAccessToken("access")
	client.SetRefreshToken("refresh")
	before := client.snapshotToken().generation
	client.SetAccessToken("access")
	client.SetRefreshToken("refresh")
	if got := client.snapshotToken().generation; got != before {
		t.Fatalf("unchanged setters incremented generation: %d -> %d", before, got)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			client.SetAccessToken("access")
		}()
		go func() {
			defer wg.Done()
			client.SetRefreshToken("refresh")
		}()
	}
	wg.Wait()
	snapshot := client.snapshotToken()
	if snapshot.accessToken != "access" || snapshot.refreshToken != "refresh" {
		t.Fatalf("token pair = %q/%q", snapshot.accessToken, snapshot.refreshToken)
	}
}

func TestIsAuthFailureCode(t *testing.T) {
	for _, code := range []int64{99, 401, 40101, 40140117} {
		if !IsAuthFailureCode(code) {
			t.Errorf("IsAuthFailureCode(%d) = false, want true", code)
		}
	}
	for _, code := range []int64{200, 400, 430004} {
		if IsAuthFailureCode(code) {
			t.Errorf("IsAuthFailureCode(%d) = true, want false", code)
		}
	}
}

func TestPassportErrorSupportsErrorsAs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, `{"code":401,"message":"invalid refresh token"}`)
	}))
	_, err := client.RefreshToken(context.Background())
	var passportErr *PassportError
	if !errors.As(err, &passportErr) {
		t.Fatalf("error = %v, want PassportError", err)
	}
	if passportErr.Code != 401 || passportErr.Message != "invalid refresh token" {
		t.Fatalf("passport error = %+v", passportErr)
	}
	if got := passportErr.Error(); got != "code: 401, message: invalid refresh token" {
		t.Fatalf("passport error text = %q", got)
	}
}
