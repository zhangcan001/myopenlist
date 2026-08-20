package sdk

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"resty.dev/v3"
)

type RefreshErrorKind string

const (
	RefreshErrorContext      RefreshErrorKind = "CONTEXT"
	RefreshErrorNetwork      RefreshErrorKind = "NETWORK"
	RefreshErrorRateLimit    RefreshErrorKind = "RATE_LIMIT"
	RefreshErrorServer       RefreshErrorKind = "SERVER"
	RefreshErrorAuthRequired RefreshErrorKind = "AUTH_REQUIRED"
	RefreshErrorPermission   RefreshErrorKind = "PERMISSION"
	RefreshErrorUnknown      RefreshErrorKind = "UNKNOWN"
	RefreshErrorSuperseded   RefreshErrorKind = "SUPERSEDED"
)

type RefreshCircuitState string

const (
	RefreshCircuitClosed       RefreshCircuitState = "CLOSED"
	RefreshCircuitOpen         RefreshCircuitState = "OPEN"
	RefreshCircuitHalfOpen     RefreshCircuitState = "HALF_OPEN"
	RefreshCircuitAuthRequired RefreshCircuitState = "AUTH_REQUIRED"
)

var ErrRefreshSuperseded = errors.New("refresh result superseded by newer token state")

type RefreshPolicy struct {
	MaxAttempts       int
	BaseBackoff       time.Duration
	MaxBackoff        time.Duration
	JitterFraction    float64
	CircuitOpenFor    time.Duration
	RateLimitFallback time.Duration
}

func DefaultRefreshPolicy() RefreshPolicy {
	return RefreshPolicy{
		MaxAttempts:       3,
		BaseBackoff:       500 * time.Millisecond,
		MaxBackoff:        4 * time.Second,
		JitterFraction:    0.20,
		CircuitOpenFor:    15 * time.Second,
		RateLimitFallback: 30 * time.Second,
	}
}

type RefreshStatus struct {
	State         RefreshCircuitState
	LastErrorKind RefreshErrorKind
	RetryAt       time.Time
}

type RefreshError struct {
	Kind    RefreshErrorKind
	State   RefreshCircuitState
	RetryAt time.Time
	Err     error
}

func (e *RefreshError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("refresh failed: %s", e.Kind)
}

func (e *RefreshError) Unwrap() error {
	return e.Err
}

func (c *Client) RefreshStatus() RefreshStatus {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return RefreshStatus{
		State:         c.refreshCircuitState,
		LastErrorKind: c.refreshLastErrorKind,
		RetryAt:       c.refreshOpenUntil,
	}
}

func classifyRefreshError(response *resty.Response, err error) RefreshErrorKind {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return RefreshErrorContext
	}

	status := 0
	if response != nil {
		status = response.StatusCode()
	}
	var passportErr *PassportError
	if errors.As(err, &passportErr) {
		if passportErr.HTTPStatus != 0 {
			status = passportErr.HTTPStatus
		}
	}
	if status == http.StatusTooManyRequests {
		return RefreshErrorRateLimit
	}
	if status >= 500 && status <= 599 {
		return RefreshErrorServer
	}
	if status == http.StatusUnauthorized {
		return RefreshErrorAuthRequired
	}
	if status == http.StatusForbidden {
		return RefreshErrorPermission
	}

	if errors.As(err, &passportErr) {
		switch {
		case passportErr.Code == 401 || passportErr.Code == 40140116:
			return RefreshErrorAuthRequired
		case passportErr.Code == 429:
			return RefreshErrorRateLimit
		case passportErr.Code >= 500 && passportErr.Code <= 599:
			return RefreshErrorServer
		case passportErr.Code == 403:
			return RefreshErrorPermission
		}
	}

	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return RefreshErrorNetwork
	}
	return RefreshErrorUnknown
}

func refreshRetryable(kind RefreshErrorKind) bool {
	return kind == RefreshErrorNetwork || kind == RefreshErrorServer
}

func (c *Client) refreshNow() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *Client) refreshSleep(ctx context.Context, delay time.Duration) error {
	if c.sleeper != nil {
		return c.sleeper(ctx, delay)
	}
	return sleepContext(ctx, delay)
}

func (c *Client) refreshRandom() float64 {
	if c.randFloat != nil {
		return c.randFloat()
	}
	return rand.Float64()
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) refreshBackoff(attempt int) time.Duration {
	policy := c.refreshPolicy
	if attempt < 1 {
		attempt = 1
	}
	delay := policy.BaseBackoff
	for i := 1; i < attempt; i++ {
		if delay >= policy.MaxBackoff/2 && policy.MaxBackoff > 0 {
			delay = policy.MaxBackoff
			break
		}
		delay *= 2
	}
	if policy.MaxBackoff > 0 && delay > policy.MaxBackoff {
		delay = policy.MaxBackoff
	}
	fraction := policy.JitterFraction
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	multiplier := 1 + (c.refreshRandom()*2-1)*fraction
	return time.Duration(float64(delay) * multiplier)
}

func retryAfterDelay(response *resty.Response, now time.Time) (time.Duration, bool) {
	if response == nil {
		return 0, false
	}
	value := strings.TrimSpace(response.Header().Get("Retry-After"))
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second, true
	}
	if retryAt, err := http.ParseTime(value); err == nil {
		if retryAt.After(now) {
			return retryAt.Sub(now), true
		}
		return 0, true
	}
	return 0, false
}

func (c *Client) resetRefreshCircuitLocked() {
	c.refreshCircuitState = RefreshCircuitClosed
	c.refreshOpenUntil = time.Time{}
	c.refreshLastErrorKind = ""
}

func (c *Client) refreshIfNeeded(ctx context.Context, generation uint64) (*RefreshTokenResp, error) {
	c.tokenMu.Lock()
	if c.tokenGeneration != generation {
		c.tokenMu.Unlock()
		return nil, nil
	}
	resp, err := c.refreshLocked(ctx)
	if errors.Is(err, ErrRefreshSuperseded) {
		return nil, nil
	}
	return resp, err
}

func (c *Client) refreshExplicit(ctx context.Context) (*RefreshTokenResp, error) {
	for {
		c.tokenMu.Lock()
		if state := c.refresh; state != nil &&
			!state.committed &&
			(state.startGeneration != c.tokenGeneration || state.refreshToken != c.refreshToken) {
			state.joined++
			c.tokenMu.Unlock()
			_, err := waitRefresh(ctx, state)
			if err != nil && !errors.Is(err, ErrRefreshSuperseded) {
				return nil, err
			}
			continue
		}
		return c.refreshLocked(ctx)
	}
}

func (c *Client) refreshLocked(ctx context.Context) (*RefreshTokenResp, error) {
	if err := ctx.Err(); err != nil {
		c.tokenMu.Unlock()
		return nil, &RefreshError{Kind: RefreshErrorContext, State: c.refreshCircuitState, Err: err}
	}
	if state := c.refresh; state != nil {
		state.joined++
		c.tokenMu.Unlock()
		return waitRefresh(ctx, state)
	}
	if err := c.refreshCircuitErrorLocked(); err != nil {
		c.tokenMu.Unlock()
		return nil, err
	}
	state := &refreshState{
		done:            make(chan struct{}),
		startGeneration: c.tokenGeneration,
		refreshToken:    c.refreshToken,
	}
	c.refresh = state
	refreshToken := state.refreshToken
	c.tokenMu.Unlock()

	resp, response, err, kind := c.runRefresh(ctx, refreshToken)
	return c.completeRefresh(state, resp, response, err, kind)
}

func waitRefresh(ctx context.Context, state *refreshState) (*RefreshTokenResp, error) {
	select {
	case <-state.done:
		return state.resp, state.err
	case <-ctx.Done():
		return nil, &RefreshError{Kind: RefreshErrorContext, Err: ctx.Err()}
	}
}

func (c *Client) refreshCircuitErrorLocked() error {
	now := c.refreshNow()
	switch c.refreshCircuitState {
	case RefreshCircuitAuthRequired:
		return &RefreshError{
			Kind:  RefreshErrorAuthRequired,
			State: RefreshCircuitAuthRequired,
			Err:   errors.New("refresh authentication required"),
		}
	case RefreshCircuitOpen:
		if now.Before(c.refreshOpenUntil) {
			return &RefreshError{
				Kind:    c.refreshLastErrorKind,
				State:   RefreshCircuitOpen,
				RetryAt: c.refreshOpenUntil,
				Err:     fmt.Errorf("refresh circuit open until %s", c.refreshOpenUntil.UTC().Format(time.RFC3339)),
			}
		}
		c.refreshCircuitState = RefreshCircuitHalfOpen
	}
	return nil
}

func (c *Client) runRefresh(ctx context.Context, refreshToken string) (*RefreshTokenResp, *resty.Response, error, RefreshErrorKind) {
	maxAttempts := c.refreshPolicy.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, response, err := c.refreshTokenRequestRaw(ctx, refreshToken)
		if err == nil {
			return resp, response, nil, ""
		}
		kind := classifyRefreshError(response, err)
		if !refreshRetryable(kind) || attempt == maxAttempts {
			return nil, response, err, kind
		}
		if err := c.refreshSleep(ctx, c.refreshBackoff(attempt)); err != nil {
			return nil, response, err, RefreshErrorContext
		}
	}
	return nil, nil, errors.New("refresh attempts exhausted"), RefreshErrorUnknown
}

func (c *Client) completeRefresh(state *refreshState, resp *RefreshTokenResp, response *resty.Response, err error, kind RefreshErrorKind) (*RefreshTokenResp, error) {
	var callback func(string, string)
	c.tokenMu.Lock()
	stale := c.tokenGeneration != state.startGeneration || c.refreshToken != state.refreshToken
	if stale {
		state.superseded = true
		state.resp = nil
		state.err = &RefreshError{
			Kind:  RefreshErrorSuperseded,
			State: c.refreshCircuitState,
			Err:   ErrRefreshSuperseded,
		}
	} else if err == nil {
		state.committed = true
		state.resp = resp
		state.err = nil
		callback = c.setTokenPairLocked(resp.AccessToken, resp.RefreshToken, true)
	} else {
		state.resp = nil
		state.err = c.recordRefreshFailureLocked(kind, response, err)
	}
	c.tokenMu.Unlock()

	if callback != nil {
		callback(resp.AccessToken, resp.RefreshToken)
	}

	c.tokenMu.Lock()
	if c.refresh == state {
		c.refresh = nil
	}
	close(state.done)
	c.tokenMu.Unlock()
	return state.resp, state.err
}

func (c *Client) recordRefreshFailureLocked(kind RefreshErrorKind, response *resty.Response, err error) error {
	now := c.refreshNow()
	state := c.refreshCircuitState
	retryAt := time.Time{}
	switch kind {
	case RefreshErrorNetwork, RefreshErrorServer:
		state = RefreshCircuitOpen
		retryAt = now.Add(c.refreshPolicy.CircuitOpenFor)
		c.refreshOpenUntil = retryAt
		c.refreshLastErrorKind = kind
	case RefreshErrorRateLimit:
		delay, ok := retryAfterDelay(response, now)
		if !ok {
			delay = c.refreshPolicy.RateLimitFallback
		}
		state = RefreshCircuitOpen
		retryAt = now.Add(delay)
		c.refreshOpenUntil = retryAt
		c.refreshLastErrorKind = kind
	case RefreshErrorAuthRequired:
		state = RefreshCircuitAuthRequired
		c.refreshOpenUntil = time.Time{}
		c.refreshLastErrorKind = kind
	case RefreshErrorContext:
		if state == RefreshCircuitHalfOpen {
			state = RefreshCircuitClosed
		}
		c.refreshLastErrorKind = kind
	default:
		if state == RefreshCircuitHalfOpen {
			state = RefreshCircuitClosed
		}
		c.refreshLastErrorKind = kind
	}
	c.refreshCircuitState = state
	return &RefreshError{Kind: kind, State: state, RetryAt: retryAt, Err: err}
}
