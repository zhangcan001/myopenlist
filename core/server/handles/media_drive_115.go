package handles

import (
	"errors"
	"net/http"

	sdk "github.com/OpenListTeam/115-sdk-go"
	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/auth115"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

var mediaDrive115Service = auth115.NewService(
	auth115.NewSDKAuthProvider(sdk.New()),
	auth115.NewCoreStorageProvisioner(),
	auth115.DefaultConfig(),
)

type mediaDrive115SessionRequest struct {
	SessionID string `json:"session_id" form:"session_id"`
}

type mediaDrive115ImportRequest struct {
	AccessToken  string `json:"access_token" form:"access_token"`
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
}

func MediaDrive115AuthCapabilities(c *gin.Context) {
	common.SuccessResp(c, mediaDrive115Service.Capabilities())
}

func MediaDrive115AuthStart(c *gin.Context) {
	result, err := mediaDrive115Service.Start(c.Request.Context())
	if err != nil {
		mediaDrive115Error(c, err)
		return
	}
	common.SuccessResp(c, result)
}

func MediaDrive115AuthStatus(c *gin.Context) {
	result, err := mediaDrive115Service.Status(c.Request.Context(), c.Query("session_id"))
	if err != nil {
		mediaDrive115Error(c, err)
		return
	}
	common.SuccessResp(c, result)
}

func MediaDrive115AuthComplete(c *gin.Context) {
	var request mediaDrive115SessionRequest
	if err := c.ShouldBind(&request); err != nil {
		common.ErrorResp(c, err, http.StatusBadRequest)
		return
	}
	result, err := mediaDrive115Service.Complete(c.Request.Context(), request.SessionID)
	if err != nil {
		mediaDrive115Error(c, err)
		return
	}
	common.SuccessResp(c, result)
}

func MediaDrive115AuthCancel(c *gin.Context) {
	var request mediaDrive115SessionRequest
	if err := c.ShouldBind(&request); err != nil {
		common.ErrorResp(c, err, http.StatusBadRequest)
		return
	}
	if err := mediaDrive115Service.Cancel(request.SessionID); err != nil {
		mediaDrive115Error(c, err)
		return
	}
	common.SuccessResp(c)
}

func MediaDrive115AuthImport(c *gin.Context) {
	var request mediaDrive115ImportRequest
	if err := c.ShouldBind(&request); err != nil {
		common.ErrorResp(c, err, http.StatusBadRequest)
		return
	}
	result, err := mediaDrive115Service.Import(c.Request.Context(), request.AccessToken, request.RefreshToken)
	if err != nil {
		mediaDrive115Error(c, err)
		return
	}
	common.SuccessResp(c, result)
}

func MediaDrive115Status(c *gin.Context) {
	common.SuccessResp(c, mediaDrive115Service.StorageStatus())
}

func MediaDrive115PersistenceRetry(c *gin.Context) {
	result, err := mediaDrive115Service.RetryPersistence(c.Request.Context())
	if err != nil {
		mediaDrive115Error(c, err)
		return
	}
	common.SuccessResp(c, result)
}

func mediaDrive115Error(c *gin.Context, err error) {
	code := http.StatusInternalServerError
	var authErr *auth115.AuthError
	if errors.As(err, &authErr) {
		switch authErr.Code {
		case auth115.CodeInvalidArgument, auth115.CodeConfigRequired, auth115.CodeTooManySessions:
			code = http.StatusBadRequest
		case auth115.CodeSessionNotFound, auth115.CodeStorageNotFound:
			code = http.StatusNotFound
		case auth115.CodeSessionExpired:
			code = http.StatusGone
		case auth115.CodeSessionCanceled, auth115.CodeStateConflict, auth115.CodeStorageConflict:
			code = http.StatusConflict
		case auth115.CodeProviderError, auth115.CodeExchangeFailed:
			code = http.StatusBadGateway
		}
	}
	// AuthError.Error deliberately contains only the stable code. Do not pass
	// provider or storage causes to the response.
	common.ErrorResp(c, err, code)
}
