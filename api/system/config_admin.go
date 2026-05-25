package system

import (
	"ad-platform/pkg/response"
	"ad-platform/pkg/util"

	"github.com/gin-gonic/gin"
)

type configSaveBody struct {
	Config  *util.Config            `json:"config"`
	Secrets *util.ConfigSecretsPatch `json:"secrets,omitempty"`
}

func (h *SystemHandler) getSystemConfig(c *gin.Context) {
	if h.configStore == nil {
		response.Error(c, response.CodeServiceUnavailable, response.MsgServiceUnavailableRetry)
		return
	}
	view, err := h.configStore.AdminGet()
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.SuccessData(c, response.MsgSuccess, view)
}

func (h *SystemHandler) putSystemConfig(c *gin.Context) {
	if h.configStore == nil {
		response.Error(c, response.CodeServiceUnavailable, response.MsgServiceUnavailableRetry)
		return
	}
	var body configSaveBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Config == nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}
	result, err := h.configStore.AdminSave(body.Config, body.Secrets)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.SuccessData(c, response.MsgSuccess, result)
}

func (h *SystemHandler) applyHotSystemConfig(c *gin.Context) {
	if h.configStore == nil {
		response.Error(c, response.CodeServiceUnavailable, response.MsgServiceUnavailableRetry)
		return
	}
	out, err := h.configStore.ApplyHotReload()
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.SuccessData(c, response.MsgSuccess, gin.H{"hot_reload": out})
}
