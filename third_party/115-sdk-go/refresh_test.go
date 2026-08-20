package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

func TestAuthRequestCoalescesConcurrentRefresh(t *testing.T) {
	var refreshCalls atomic.Int32
	var apiCalls atomic.Int32
	var callbackCalls atomic.Int32
	var releaseRefresh = make(chan struct{})
	var allInitial401 = make(chan struct{})
	var closeInitial401Once sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/open/refreshToken":
			if refreshCalls.Add(1) != 1 {
				t.Errorf("refresh called more than once")
			}
			<-releaseRefresh
			fmt.Fprint(w, `{"code":0,"data":{"access_token":"new-access","refresh_token":"new-refresh"}}`)
		case "/open/user/info":
			if r.Header.Get("Authorization") == "Bearer old-access" {
				if apiCalls.Add(1) == 3 {
					closeInitial401Once.Do(func() { close(allInitial401) })
				}
				fmt.Fprint(w, `{"state":false,"code":40101,"message":"expired"}`)
				return
			}
			if r.Header.Get("Authorization") != "Bearer new-access" {
				t.Errorf("retry used unexpected token: %q", r.Header.Get("Authorization"))
			}
			fmt.Fprint(w, `{"state":true,"data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := New(WithAccessToken("old-access"), WithRefreshToken("old-refresh"))
	client.SetHttpClient(&http.Client{Transport: rewriteTransport{base: base}})
	client.SetOnRefreshToken(func(string, string) { callbackCalls.Add(1) })
	_, initialGeneration := client.tokenSnapshot()

	const requestCount = 3
	results := make(chan error, requestCount)
	var wg sync.WaitGroup
	for range requestCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, requestErr := client.AuthRequest(context.Background(), ApiUserInfo, http.MethodGet, nil)
			results <- requestErr
		}()
	}

	select {
	case <-allInitial401:
	case <-time.After(2 * time.Second):
		t.Fatal("requests did not all reach the initial 401")
	}
	close(releaseRefresh)
	wg.Wait()
	close(results)

	for requestErr := range results {
		if requestErr != nil {
			t.Fatalf("request failed: %v", requestErr)
		}
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls = %d, want 1", got)
	}
	if got := callbackCalls.Load(); got != 1 {
		t.Fatalf("refresh callbacks = %d, want 1", got)
	}
	client.tokenMu.RLock()
	finalGeneration := client.tokenGeneration
	client.tokenMu.RUnlock()
	if finalGeneration != initialGeneration+1 {
		t.Fatalf("token generation = %d, want %d", finalGeneration, initialGeneration+1)
	}
}

func TestAuthRequestRefreshFailureIsShared(t *testing.T) {
	var refreshCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/refreshToken") {
			refreshCalls.Add(1)
			fmt.Fprint(w, `{"code":401,"message":"invalid refresh token"}`)
			return
		}
		fmt.Fprint(w, `{"state":false,"code":40101,"message":"expired"}`)
	}))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := New(WithAccessToken("old-access"), WithRefreshToken("old-refresh"))
	client.SetHttpClient(&http.Client{Transport: rewriteTransport{base: base}})

	const requestCount = 3
	results := make(chan error, requestCount)
	var wg sync.WaitGroup
	for range requestCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, requestErr := client.AuthRequest(context.Background(), ApiUserInfo, http.MethodGet, nil)
			results <- requestErr
		}()
	}
	wg.Wait()
	close(results)

	for requestErr := range results {
		if requestErr == nil || !strings.Contains(requestErr.Error(), "invalid refresh token") {
			t.Fatalf("unexpected request error: %v", requestErr)
		}
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls = %d, want 1", got)
	}
}
