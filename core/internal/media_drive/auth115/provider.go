package auth115

import (
	"context"
	"strconv"

	sdk "github.com/OpenListTeam/115-sdk-go"
)

type SDKAuthProvider struct {
	client *sdk.Client
}

func NewSDKAuthProvider(client *sdk.Client) *SDKAuthProvider {
	if client == nil {
		client = sdk.New()
	}
	return &SDKAuthProvider{client: client}
}

func (p *SDKAuthProvider) StartDeviceCode(ctx context.Context, clientID, codeVerifier string) (*DeviceCodeResult, error) {
	result, err := p.client.AuthDeviceCode(ctx, clientID, codeVerifier)
	if err != nil {
		return nil, err
	}
	return &DeviceCodeResult{
		UID:    result.UID,
		Time:   result.Time,
		QRCode: result.QrCode,
		Sign:   result.Sign,
	}, nil
}

func (p *SDKAuthProvider) GetQRStatus(ctx context.Context, uid string, timestamp int64, sign string) (*QRStatusResult, error) {
	result, err := p.client.QrCodeStatus(ctx, uid, strconv.FormatInt(timestamp, 10), sign)
	if err != nil {
		return nil, err
	}
	return &QRStatusResult{
		Status:  result.Status,
		Message: result.Msg,
		Version: result.Version,
	}, nil
}

func (p *SDKAuthProvider) ExchangeCode(ctx context.Context, uid, codeVerifier string) (*TokenPairResult, error) {
	result, err := p.client.CodeToToken(ctx, uid, codeVerifier)
	if err != nil {
		return nil, err
	}
	return &TokenPairResult{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}
