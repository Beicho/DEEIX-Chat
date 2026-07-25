package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	appauth "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/auth"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// CreateInvitationCodesRequest 批量生成邀请码请求。
type CreateInvitationCodesRequest struct {
	Label     string     `json:"label" binding:"omitempty,max=80"`
	Count     int        `json:"count" binding:"omitempty,min=1,max=1000"`
	MaxUses   int        `json:"maxUses" binding:"omitempty,min=1,max=100000"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

// UpdateInvitationCodeRequest 启用/停用邀请码请求。
type UpdateInvitationCodeRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// InvitationCodeResponse 邀请码响应。
type InvitationCodeResponse struct {
	PublicID   string     `json:"publicID"`
	Code       string     `json:"code,omitempty"`
	Label      string     `json:"label"`
	MaxUses    int        `json:"maxUses"`
	UsedCount  int        `json:"usedCount"`
	Enabled    bool       `json:"enabled"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// InvitationCodeListResponse 邀请码列表响应。
type InvitationCodeListResponse struct {
	Results []InvitationCodeResponse `json:"results"`
	Total   int                      `json:"total"`
}

func toInvitationCodeResponse(item appauth.InvitationCodeResult) InvitationCodeResponse {
	return InvitationCodeResponse{
		PublicID:   item.PublicID,
		Code:       item.Code,
		Label:      item.Label,
		MaxUses:    item.MaxUses,
		UsedCount:  item.UsedCount,
		Enabled:    item.Enabled,
		ExpiresAt:  item.ExpiresAt,
		LastUsedAt: item.LastUsedAt,
		CreatedAt:  item.CreatedAt,
	}
}

// ListInvitationCodes 列出全部邀请码（不含明文）。
func (h *Handler) ListInvitationCodes(c *gin.Context) {
	items, err := h.service.ListInvitationCodes(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "list invitation codes failed")
		return
	}
	results := make([]InvitationCodeResponse, 0, len(items))
	for _, item := range items {
		results = append(results, toInvitationCodeResponse(item))
	}
	response.Success(c, InvitationCodeListResponse{Results: results, Total: len(results)})
}

// CreateInvitationCodes 批量生成邀请码，明文仅在本次响应返回。
func (h *Handler) CreateInvitationCodes(c *gin.Context) {
	var req CreateInvitationCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	actorID := middleware.MustUserID(c)
	items, err := h.service.CreateInvitationCodes(
		c.Request.Context(),
		actorID,
		req.Label,
		req.Count,
		req.MaxUses,
		req.ExpiresAt,
	)
	if err != nil {
		response.ErrorFrom(c, http.StatusBadRequest, err)
		return
	}
	results := make([]InvitationCodeResponse, 0, len(items))
	for _, item := range items {
		results = append(results, toInvitationCodeResponse(item))
	}
	h.recordAudit(c, actorID, "invitation_code.create", "invitation_code", "", gin.H{
		"count":   len(results),
		"label":   strings.TrimSpace(req.Label),
		"maxUses": req.MaxUses,
	})
	response.Success(c, InvitationCodeListResponse{Results: results, Total: len(results)})
}

// UpdateInvitationCode 启用或停用一个邀请码。
func (h *Handler) UpdateInvitationCode(c *gin.Context) {
	var req UpdateInvitationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	actorID := middleware.MustUserID(c)
	publicID := strings.TrimSpace(c.Param("code_id"))
	item, err := h.service.SetInvitationCodeEnabled(c.Request.Context(), publicID, *req.Enabled, actorID)
	if err != nil {
		response.ErrorFrom(c, http.StatusBadRequest, err)
		return
	}
	h.recordAudit(c, actorID, "invitation_code.update", "invitation_code", publicID, gin.H{
		"enabled": *req.Enabled,
	})
	response.Success(c, toInvitationCodeResponse(*item))
}

// ExportInvitationCodes 批量生成并直接下载 CSV，明文只在下载文件中出现一次。
func (h *Handler) ExportInvitationCodes(c *gin.Context) {
	var req CreateInvitationCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	actorID := middleware.MustUserID(c)
	items, err := h.service.CreateInvitationCodes(
		c.Request.Context(),
		actorID,
		req.Label,
		req.Count,
		req.MaxUses,
		req.ExpiresAt,
	)
	if err != nil {
		response.ErrorFrom(c, http.StatusBadRequest, err)
		return
	}
	h.recordAudit(c, actorID, "invitation_code.export", "invitation_code", "", gin.H{
		"count": len(items),
		"label": strings.TrimSpace(req.Label),
	})

	var builder strings.Builder
	builder.WriteString("\ufeffcode,label,max_uses,expires_at\n")
	for _, item := range items {
		expires := ""
		if item.ExpiresAt != nil {
			expires = item.ExpiresAt.UTC().Format(time.RFC3339)
		}
		builder.WriteString(fmt.Sprintf(
			"%s,%s,%d,%s\n",
			csvField(item.Code),
			csvField(item.Label),
			item.MaxUses,
			csvField(expires),
		))
	}
	filename := fmt.Sprintf("invitation-codes-%s.csv", time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(builder.String()))
}

func csvField(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.ContainsAny(trimmed, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(trimmed, "\"", "\"\"") + "\""
	}
	return trimmed
}
