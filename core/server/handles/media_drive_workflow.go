package handles

import (
	"errors"
	"io"
	"net/http"

	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/mount"
	"github.com/OpenListTeam/OpenList/v4/internal/media_drive/workflow"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

var mediaDriveWorkflowManager = workflow.NewManager(
	mediaDrive115Service,
	mediaDriveWebDAVManager,
	mount.NewManager(mount.NewSettingProfileStore()),
)

type mediaDriveWorkflowStartRequest struct {
	WebDAVUsername string `json:"webdav_username"`
	WebDAVPassword string `json:"webdav_password"`
}

type mediaDriveMountProfileRequest struct {
	DriveLetter   string `json:"drive_letter"`
	WebDAVURL     string `json:"webdav_url"`
	Enabled       *bool  `json:"enabled"`
	AutoReconnect *bool  `json:"auto_reconnect"`
}

func MediaDriveWorkflowStatus(c *gin.Context) {
	common.SuccessResp(c, mediaDriveWorkflowManager.Status())
}

func MediaDriveWorkflowStart(c *gin.Context) {
	var request mediaDriveWorkflowStartRequest
	if err := c.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		common.ErrorResp(c, err, http.StatusBadRequest)
		return
	}
	if err := mediaDriveWorkflowManager.StartWorkflow(c.Request.Context(), workflow.StartOptions{
		WebDAVUsername: request.WebDAVUsername,
		WebDAVPassword: request.WebDAVPassword,
	}); err != nil {
		mediaDriveWorkflowError(c, err)
		return
	}
	common.SuccessResp(c, mediaDriveWorkflowManager.Status())
}

func MediaDriveWorkflowStop(c *gin.Context) {
	if err := mediaDriveWorkflowManager.StopWorkflow(c.Request.Context()); err != nil {
		mediaDriveWorkflowError(c, err)
		return
	}
	common.SuccessResp(c, mediaDriveWorkflowManager.Status())
}

func MediaDriveWorkflowHealth(c *gin.Context) {
	common.SuccessResp(c, mediaDriveWorkflowManager.Health())
}

func MediaDriveMountProfile(c *gin.Context) {
	profile, err := mediaDriveWorkflowManager.MountProfile()
	if err != nil {
		mediaDriveWorkflowError(c, err)
		return
	}
	common.SuccessResp(c, profile)
}

func MediaDriveMountProfileUpdate(c *gin.Context) {
	var request mediaDriveMountProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ErrorResp(c, err, http.StatusBadRequest)
		return
	}
	profile, err := mediaDriveWorkflowManager.MountProfile()
	if err != nil {
		mediaDriveWorkflowError(c, err)
		return
	}
	if request.DriveLetter != "" {
		profile.DriveLetter = request.DriveLetter
	}
	if request.WebDAVURL != "" {
		profile.WebDAVURL = request.WebDAVURL
	}
	if request.Enabled != nil {
		profile.Enabled = *request.Enabled
	}
	if request.AutoReconnect != nil {
		profile.AutoReconnect = *request.AutoReconnect
	}
	if err := mediaDriveWorkflowManager.UpdateMountProfile(profile); err != nil {
		mediaDriveWorkflowError(c, err)
		return
	}
	profile, err = mediaDriveWorkflowManager.MountProfile()
	if err != nil {
		mediaDriveWorkflowError(c, err)
		return
	}
	common.SuccessResp(c, profile)
}

func mediaDriveWorkflowError(c *gin.Context, err error) {
	code := http.StatusInternalServerError
	if errors.Is(err, workflow.ErrWorkflowRunning) {
		code = http.StatusConflict
	}
	common.ErrorWithDataResp(c, err, code, mediaDriveWorkflowManager.Status())
}
