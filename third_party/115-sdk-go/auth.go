// docs: https://www.yuque.com/115yun/open/shtpzfhewv5nag11
package sdk

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
)

type AuthDeviceCodeResp struct {
	UID    string `json:"uid"`
	Time   int64  `json:"time"`
	QrCode string `json:"qrcode"`
	Sign   string `json:"sign"`
}

// $code_challenge = base64_encode(sha256($code_verifier));
func calCodeChanllenge(codeVerifier string) string {
	sha := sha256.New()
	sha.Write([]byte(codeVerifier))
	return base64.StdEncoding.EncodeToString(sha.Sum(nil))
}

func (c *Client) AuthDeviceCode(ctx context.Context, clientID string, codeVerifier string) (*AuthDeviceCodeResp, error) {
	var resp AuthDeviceCodeResp
	_, err := c.passportRequest(ctx, ApiAuthDeviceCode, http.MethodPost, &resp, ReqWithForm(Form{
		"client_id":             clientID,
		"code_challenge":        calCodeChanllenge(codeVerifier),
		"code_challenge_method": "sha256",
	}))
	if err != nil {
		return nil, err
	}
	return &resp, err
}

type QrCodeStatusResp struct {
	Msg     string `json:"msg"`
	Status  int    `json:"status"`
	Version string `json:"version"`
}

func (c *Client) QrCodeStatus(ctx context.Context, uid, time, sign string) (*QrCodeStatusResp, error) {
	var resp QrCodeStatusResp
	_, err := c.passportRequest(ctx, ApiQrCodeStatus, http.MethodGet, &resp, ReqWithQuery(Form{
		"uid":  uid,
		"time": time,
		"sign": sign,
	}))
	if err != nil {
		return nil, err
	}
	return &resp, err
}

type CodeToTokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (c *Client) CodeToToken(ctx context.Context, uid, codeVerifier string) (*CodeToTokenResp, error) {
	var resp CodeToTokenResp
	_, err := c.passportRequest(ctx, ApiCodeToToken, http.MethodPost, &resp, ReqWithForm(Form{
		"uid":           uid,
		"code_verifier": codeVerifier,
	}))
	if err != nil {
		return nil, err
	}
	// QR token exchange preserves the existing no-callback behavior.
	c.setTokenPair(resp.AccessToken, resp.RefreshToken, false)
	return &resp, err
}

type RefreshTokenResp CodeToTokenResp

func (c *Client) RefreshToken(ctx context.Context) (*RefreshTokenResp, error) {
	return c.refreshExplicit(ctx)
}

type refreshState struct {
	done   chan struct{}
	resp   *RefreshTokenResp
	err    error
	joined int
}

func (c *Client) refreshIfNeeded(ctx context.Context, generation uint64) (*RefreshTokenResp, error) {
	c.tokenMu.Lock()
	if c.tokenGeneration != generation {
		c.tokenMu.Unlock()
		return nil, nil
	}
	return c.refreshLocked(ctx)
}

func (c *Client) refreshExplicit(ctx context.Context) (*RefreshTokenResp, error) {
	c.tokenMu.Lock()
	return c.refreshLocked(ctx)
}

func (c *Client) refreshLocked(ctx context.Context) (*RefreshTokenResp, error) {
	if state := c.refresh; state != nil {
		state.joined++
		c.tokenMu.Unlock()
		select {
		case <-state.done:
			return state.resp, state.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	state := &refreshState{done: make(chan struct{})}
	c.refresh = state
	refreshToken := c.refreshToken
	c.tokenMu.Unlock()

	resp, err := c.refreshTokenRequest(ctx, refreshToken)

	c.tokenMu.Lock()
	state.resp = resp
	state.err = err
	c.refresh = nil
	close(state.done)
	c.tokenMu.Unlock()
	return resp, err
}

func (c *Client) refreshTokenRequest(ctx context.Context, refreshToken string) (*RefreshTokenResp, error) {
	var resp RefreshTokenResp
	_, err := c.passportRequest(ctx, ApiRefreshToken, http.MethodPost, &resp, ReqWithForm(Form{
		"refresh_token": refreshToken,
	}))
	if err != nil {
		return nil, err
	}
	c.setTokenPair(resp.AccessToken, resp.RefreshToken, true)
	return &resp, err
}
