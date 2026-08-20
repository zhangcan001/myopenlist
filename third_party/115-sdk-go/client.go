package sdk

import (
	"context"
	"net/http"
	"sync"

	"resty.dev/v3"
)

type tokenSnapshot struct {
	accessToken  string
	refreshToken string
	generation   uint64
}

type Client struct {
	client *resty.Client

	tokenMu         sync.RWMutex
	accessToken     string
	refreshToken    string
	tokenGeneration uint64
	refresh         *refreshState
	onRefreshToken  func(string, string)
}

func New(opts ...Option) *Client {
	c := &Client{
		client: resty.New(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func Default() *Client {
	return New()
}

func (w *Client) SetHttpClient(httpClient *http.Client) *Client {
	w.client = resty.NewWithClient(httpClient)
	return w
}

func (w *Client) SetUserAgent(userAgent string) *Client {
	w.client.SetHeader("User-Agent", userAgent)
	return w
}

func (w *Client) SetDebug(d bool) *Client {
	w.client.SetDebug(d)
	return w
}

func (w *Client) EnableTrace() *Client {
	w.client.EnableTrace()
	return w
}

func (w *Client) SetProxy(proxy string) *Client {
	w.client.SetProxy(proxy)
	return w
}

func (w *Client) SetAccessToken(token string) *Client {
	w.tokenMu.Lock()
	if w.accessToken != token {
		w.accessToken = token
		w.tokenGeneration++
	}
	w.tokenMu.Unlock()
	return w
}

func (w *Client) SetRefreshToken(token string) *Client {
	w.tokenMu.Lock()
	if w.refreshToken != token {
		w.refreshToken = token
		w.tokenGeneration++
	}
	w.tokenMu.Unlock()
	return w
}

func (w *Client) SetOnRefreshToken(fn func(accessToken string, refreshToken string)) *Client {
	w.tokenMu.Lock()
	w.onRefreshToken = fn
	w.tokenMu.Unlock()
	return w
}

func (w *Client) snapshotToken() tokenSnapshot {
	w.tokenMu.RLock()
	defer w.tokenMu.RUnlock()
	return tokenSnapshot{
		accessToken:  w.accessToken,
		refreshToken: w.refreshToken,
		generation:   w.tokenGeneration,
	}
}

// Internal authentication flows that rotate both tokens must use this atomic pair update.
func (w *Client) setTokenPair(accessToken, refreshToken string, notify bool) {
	w.tokenMu.Lock()
	w.accessToken = accessToken
	w.refreshToken = refreshToken
	w.tokenGeneration++
	callback := w.onRefreshToken
	w.tokenMu.Unlock()

	if notify && callback != nil {
		callback(accessToken, refreshToken)
	}
}

func (w *Client) NewRequest(ctx context.Context) *resty.Request {
	return w.client.R().SetContext(ctx)
}
