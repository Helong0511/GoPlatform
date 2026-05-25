package system

import (
	"net/url"
	"strings"

	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

type ipBanRequest struct {
	IP     string `json:"ip"`
	Reason string `json:"reason"`
}

type ipWhitelistRequest struct {
	IP     string `json:"ip"`
	Remark string `json:"remark"`
}

func (h *SystemHandler) ipAccessGuard(c *gin.Context) bool {
	if h.ipGuard == nil {
		response.Error(c, response.CodeServiceUnavailable, response.MsgServiceUnavailableRetry)
		return false
	}
	return true
}

func (h *SystemHandler) getIPAccessStatus(c *gin.Context) {
	if !h.ipAccessGuard(c) {
		return
	}
	response.SuccessData(c, response.MsgSuccess, h.ipGuard.Status())
}

func (h *SystemHandler) listIPBlacklist(c *gin.Context) {
	if !h.ipAccessGuard(c) {
		return
	}
	response.SuccessData(c, response.MsgSuccess, gin.H{
		"entries": h.ipGuard.ListBanned(),
	})
}

func (h *SystemHandler) banIP(c *gin.Context) {
	if !h.ipAccessGuard(c) {
		return
	}
	var req ipBanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}
	if err := h.ipGuard.Ban(strings.TrimSpace(req.IP), strings.TrimSpace(req.Reason)); err != nil {
		if strings.Contains(err.Error(), "whitelisted") {
			response.BadRequest(c, response.MsgIPWhitelistedCannotBan)
			return
		}
		response.BadRequest(c, response.MsgInvalidIPAddress)
		return
	}
	response.Success(c, response.MsgSuccess)
}

func (h *SystemHandler) unbanIP(c *gin.Context) {
	if !h.ipAccessGuard(c) {
		return
	}
	ip := pathUnescapeIPParam(c.Param("ip"))
	if ip == "" {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}
	if !h.ipGuard.Unban(ip) {
		response.NotFound(c, response.MsgIPNotInBlacklist)
		return
	}
	response.Success(c, response.MsgSuccess)
}

func (h *SystemHandler) listIPWhitelist(c *gin.Context) {
	if !h.ipAccessGuard(c) {
		return
	}
	response.SuccessData(c, response.MsgSuccess, gin.H{
		"entries": h.ipGuard.ListWhitelist(),
	})
}

func (h *SystemHandler) addIPWhitelist(c *gin.Context) {
	if !h.ipAccessGuard(c) {
		return
	}
	var req ipWhitelistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}
	if err := h.ipGuard.AddWhitelist(strings.TrimSpace(req.IP), strings.TrimSpace(req.Remark)); err != nil {
		response.BadRequest(c, response.MsgInvalidIPAddress)
		return
	}
	response.Success(c, response.MsgSuccess)
}

func (h *SystemHandler) removeIPWhitelist(c *gin.Context) {
	if !h.ipAccessGuard(c) {
		return
	}
	ip := pathUnescapeIPParam(c.Param("ip"))
	if ip == "" {
		response.BadRequest(c, response.MsgInvalidRequestData)
		return
	}
	if !h.ipGuard.RemoveWhitelist(ip) {
		response.NotFound(c, response.MsgIPNotInWhitelist)
		return
	}
	response.Success(c, response.MsgSuccess)
}

func pathUnescapeIPParam(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if decoded, err := url.PathUnescape(raw); err == nil {
		return strings.TrimSpace(decoded)
	}
	return raw
}
