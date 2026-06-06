package security

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	appsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/security"
	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	powService         *appsecurity.PoWService
	proofService       *appsecurity.RequestProofService
	fingerprintService *appsecurity.FingerprintService
}

func NewHandler(powService *appsecurity.PoWService, proofService *appsecurity.RequestProofService, fingerprintService *appsecurity.FingerprintService) *Handler {
	return &Handler{
		powService:         powService,
		proofService:       proofService,
		fingerprintService: fingerprintService,
	}
}

func (h *Handler) GetPoWChallenge(c *gin.Context) {
	if h == nil || h.powService == nil {
		response.Error(c, http.StatusServiceUnavailable, "pow is unavailable")
		return
	}
	action := strings.TrimSpace(c.Query("action"))
	if action == "" {
		action = "default"
	}
	challenge, err := h.powService.GenerateChallenge(c.Request.Context(), middleware.MustUserID(c), action)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "generate pow challenge failed")
		return
	}
	response.Success(c, PoWChallengeResponse{
		Challenge:  challenge.Challenge,
		Difficulty: challenge.Difficulty,
		Action:     challenge.Action,
		ExpiresAt:  challenge.ExpiresAt,
	})
}

type PoWChallengeResponse struct {
	Challenge  string    `json:"challenge"`
	Difficulty int       `json:"difficulty"`
	Action     string    `json:"action"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type BootstrapBrowserKeyRequest struct {
	SessionID    string `json:"sessionId" binding:"omitempty,max=128"`
	KeyID        string `json:"keyId" binding:"omitempty,max=128"`
	PublicKeyJWK string `json:"publicKeyJwk" binding:"required,max=4096"`
}

type BootstrapBrowserKeyResponse struct {
	KeyID string `json:"keyId"`
}

func (h *Handler) BootstrapBrowserKey(c *gin.Context) {
	if h == nil || h.proofService == nil {
		response.Error(c, http.StatusServiceUnavailable, "browser proof is unavailable")
		return
	}
	var req BootstrapBrowserKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	sessionID := middleware.MustSessionID(c)
	if strings.TrimSpace(req.SessionID) != "" && strings.TrimSpace(req.SessionID) != sessionID {
		response.Error(c, http.StatusForbidden, "session mismatch")
		return
	}
	keyID := strings.TrimSpace(req.KeyID)
	if keyID == "" {
		keyID = appsecurity.BrowserKeyID(req.PublicKeyJWK)
	}
	if err := h.proofService.BootstrapBrowserKey(c.Request.Context(), middleware.MustUserID(c), sessionID, keyID, req.PublicKeyJWK); err != nil {
		response.Error(c, http.StatusForbidden, "browser key is invalid")
		return
	}
	response.Success(c, BootstrapBrowserKeyResponse{KeyID: keyID})
}

type DeviceFingerprintRequest struct {
	FingerprintID       string  `json:"fingerprintId" binding:"omitempty,max=128"`
	ScreenResolution    string  `json:"screenResolution" binding:"omitempty,max=50"`
	ColorDepth          int     `json:"colorDepth"`
	PixelRatio          float64 `json:"pixelRatio"`
	HardwareConcurrency int     `json:"hardwareConcurrency"`
	DeviceMemory        int     `json:"deviceMemory"`
	MaxTouchPoints      int     `json:"maxTouchPoints"`
	UserAgent           string  `json:"userAgent" binding:"omitempty,max=1024"`
	Language            string  `json:"language" binding:"omitempty,max=32"`
	Timezone            string  `json:"timezone" binding:"omitempty,max=80"`
	Platform            string  `json:"platform" binding:"omitempty,max=80"`
	CanvasHash          string  `json:"canvasHash" binding:"omitempty,max=128"`
	WebGLVendor         string  `json:"webglVendor" binding:"omitempty,max=255"`
	WebGLRenderer       string  `json:"webglRenderer" binding:"omitempty,max=255"`
	FontsHash           string  `json:"fontsHash" binding:"omitempty,max=128"`
	AudioHash           string  `json:"audioHash" binding:"omitempty,max=128"`
}

type DeviceFingerprintResponse struct {
	FingerprintID string  `json:"fingerprintId"`
	RiskLevel     string  `json:"riskLevel,omitempty"`
	Confidence    float64 `json:"confidence,omitempty"`
}

func (h *Handler) RecordFingerprint(c *gin.Context) {
	if h == nil || h.fingerprintService == nil {
		response.Error(c, http.StatusServiceUnavailable, "fingerprint service is unavailable")
		return
	}
	var req DeviceFingerprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	fp := domainsecurity.DeviceFingerprint{
		UserID:              middleware.MustUserID(c),
		FingerprintID:       strings.TrimSpace(req.FingerprintID),
		ScreenResolution:    strings.TrimSpace(req.ScreenResolution),
		ColorDepth:          req.ColorDepth,
		PixelRatio:          req.PixelRatio,
		HardwareConcurrency: req.HardwareConcurrency,
		DeviceMemory:        req.DeviceMemory,
		MaxTouchPoints:      req.MaxTouchPoints,
		UserAgent:           firstNonEmpty(strings.TrimSpace(req.UserAgent), c.Request.UserAgent()),
		Language:            strings.TrimSpace(req.Language),
		Timezone:            strings.TrimSpace(req.Timezone),
		Platform:            strings.TrimSpace(req.Platform),
		CanvasHash:          strings.TrimSpace(req.CanvasHash),
		WebGLVendor:         strings.TrimSpace(req.WebGLVendor),
		WebGLRenderer:       strings.TrimSpace(req.WebGLRenderer),
		FontsHash:           strings.TrimSpace(req.FontsHash),
		AudioHash:           strings.TrimSpace(req.AudioHash),
		IPAddress:           c.ClientIP(),
	}
	detection, err := h.fingerprintService.RecordFingerprint(c.Request.Context(), fp)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "record fingerprint failed")
		return
	}
	fingerprintID := appsecurity.StableFingerprintID(fp)
	result := DeviceFingerprintResponse{FingerprintID: fingerprintID}
	if detection != nil {
		result.RiskLevel = detection.RiskLevel
		result.Confidence = detection.ConfidenceScore
	}
	response.Success(c, result)
}

type FingerprintAssociationResponse struct {
	ID              uint      `json:"id"`
	FingerprintID   string    `json:"fingerprintId"`
	UserIDs         []uint    `json:"userIds"`
	ConfidenceScore float64   `json:"confidenceScore"`
	RiskLevel       string    `json:"riskLevel"`
	DetectedAt      time.Time `json:"detectedAt"`
}

func (h *Handler) ListFingerprintAssociations(c *gin.Context) {
	if h == nil || h.fingerprintService == nil {
		response.Error(c, http.StatusServiceUnavailable, "fingerprint service is unavailable")
		return
	}
	page, pageSize := pageParams(c)
	items, total, err := h.fingerprintService.ListAssociations(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "list fingerprint associations failed")
		return
	}
	result := make([]FingerprintAssociationResponse, 0, len(items))
	for _, item := range items {
		result = append(result, FingerprintAssociationResponse{
			ID:              item.ID,
			FingerprintID:   item.FingerprintID,
			UserIDs:         item.UserIDs,
			ConfidenceScore: item.ConfidenceScore,
			RiskLevel:       item.RiskLevel,
			DetectedAt:      item.DetectedAt,
		})
	}
	response.SuccessPage(c, total, result)
}

func pageParams(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	return page, pageSize
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
