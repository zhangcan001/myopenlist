package handles

import (
	"errors"
	"net/http"

	managedwebdav "github.com/OpenListTeam/OpenList/v4/internal/media_drive/webdav"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

var mediaDriveWebDAVManager = managedwebdav.NewManager(managedwebdav.NewSettingProfileStore())

type mediaDriveWebDAVProfileRequest struct {
	Enabled            *bool  `json:"enabled"`
	BindAddress        string `json:"bind_address"`
	Port               *int   `json:"port"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	AllowLocalhostOnly *bool  `json:"allow_localhost_only"`
}

func MediaDriveWebDAVProfile(c *gin.Context) {
	profile, err := mediaDriveWebDAVManager.Profile()
	if err != nil {
		mediaDriveWebDAVError(c, err)
		return
	}
	common.SuccessResp(c, profile)
}

func MediaDriveWebDAVSaveProfile(c *gin.Context) {
	var request mediaDriveWebDAVProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ErrorResp(c, err, http.StatusBadRequest)
		return
	}
	profile, err := mediaDriveWebDAVManager.UpdateProfile(managedwebdav.ProfileUpdate{
		Enabled:            request.Enabled,
		BindAddress:        request.BindAddress,
		Port:               request.Port,
		Username:           request.Username,
		Password:           request.Password,
		AllowLocalhostOnly: request.AllowLocalhostOnly,
	})
	if err != nil {
		mediaDriveWebDAVError(c, err)
		return
	}
	common.SuccessResp(c, profile)
}

func MediaDriveWebDAVStart(c *gin.Context) {
	if err := mediaDriveWebDAVManager.Start(); err != nil {
		mediaDriveWebDAVError(c, err)
		return
	}
	status, err := mediaDriveWebDAVManager.Status()
	if err != nil {
		mediaDriveWebDAVError(c, err)
		return
	}
	common.SuccessResp(c, status)
}

func MediaDriveWebDAVStop(c *gin.Context) {
	if err := mediaDriveWebDAVManager.Stop(); err != nil {
		mediaDriveWebDAVError(c, err)
		return
	}
	status, err := mediaDriveWebDAVManager.Status()
	if err != nil {
		mediaDriveWebDAVError(c, err)
		return
	}
	common.SuccessResp(c, status)
}

func MediaDriveWebDAVStatus(c *gin.Context) {
	status, err := mediaDriveWebDAVManager.Status()
	if err != nil {
		mediaDriveWebDAVError(c, err)
		return
	}
	common.SuccessResp(c, status)
}

func mediaDriveWebDAVError(c *gin.Context, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, managedwebdav.ErrInvalidProfile),
		errors.Is(err, managedwebdav.ErrInvalidPort),
		errors.Is(err, managedwebdav.ErrLocalhostOnlyRequired),
		errors.Is(err, managedwebdav.ErrPasswordNotConfigured),
		errors.Is(err, managedwebdav.ErrProfileDisabled):
		code = http.StatusBadRequest
	case errors.Is(err, managedwebdav.ErrServiceRunning),
		errors.Is(err, managedwebdav.ErrPortConflict):
		code = http.StatusConflict
	}
	common.ErrorResp(c, err, code)
}
