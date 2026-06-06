package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	appsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/security"
	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

const BrowserProofHeader = "X-DEEIX-Proof"

type BrowserProofVerifyInput = appsecurity.RequestProofVerifyInput

type BrowserProofVerifier interface {
	VerifyRequestProof(ctx context.Context, input BrowserProofVerifyInput) error
}

type protectedBrowserProofRoute struct {
	method  string
	pattern *regexp.Regexp
	action  string
}

var protectedBrowserProofRoutes = []protectedBrowserProofRoute{
	{method: http.MethodPost, pattern: regexp.MustCompile(`^/api/v1/conversations/[^/]+/messages$`), action: "send_message"},
	{method: http.MethodPost, pattern: regexp.MustCompile(`^/api/v1/conversations/[^/]+/messages/stream$`), action: "send_message"},
	{method: http.MethodPost, pattern: regexp.MustCompile(`^/api/v1/conversations/[^/]+/media/images/generations/stream$`), action: "generate_image"},
	{method: http.MethodPost, pattern: regexp.MustCompile(`^/api/v1/conversations/[^/]+/media/images/edits/stream$`), action: "generate_image"},
	{method: http.MethodPost, pattern: regexp.MustCompile(`^/api/v1/files$`), action: "upload_file"},
	{method: http.MethodPost, pattern: regexp.MustCompile(`^/api/v1/conversation-runs/[^/]+/cancel$`), action: "cancel_generation"},
}

func BrowserProofMiddleware(verifier BrowserProofVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		action, ok := browserProofAction(c.Request.Method, c.Request.URL.Path)
		if !ok {
			c.Next()
			return
		}
		if verifier == nil {
			response.Error(c, http.StatusForbidden, "browser proof verifier unavailable")
			c.Abort()
			return
		}
		userID := MustUserID(c)
		sessionID := MustSessionID(c)
		if userID == 0 || strings.TrimSpace(sessionID) == "" {
			response.Error(c, http.StatusUnauthorized, "missing authenticated session")
			c.Abort()
			return
		}
		proofHeader := strings.TrimSpace(c.GetHeader(BrowserProofHeader))
		if proofHeader == "" {
			response.Error(c, http.StatusForbidden, "browser proof is required")
			c.Abort()
			return
		}
		var proofPayload browserProofPayload
		if err := json.Unmarshal([]byte(proofHeader), &proofPayload); err != nil {
			response.Error(c, http.StatusForbidden, "browser proof is invalid")
			c.Abort()
			return
		}
		proof := proofPayload.toDomain()
		body, err := readAndRestoreRequestBody(c)
		if err != nil {
			response.ErrorWithCode(c, http.StatusBadRequest, "browser_proof.body_read_failed", "read request body failed")
			c.Abort()
			return
		}
		if err = verifier.VerifyRequestProof(c.Request.Context(), appsecurity.RequestProofVerifyInput{
			UserID:    userID,
			SessionID: sessionID,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Query:     c.Request.URL.RawQuery,
			Action:    action,
			Body:      body,
			Proof:     &proof,
		}); err != nil {
			response.Error(c, http.StatusForbidden, "browser proof is invalid")
			c.Abort()
			return
		}
		c.Next()
	}
}

type browserProofPayload struct {
	KeyID     string           `json:"keyId"`
	SessionID string           `json:"sessionId"`
	Timestamp int64            `json:"timestamp"`
	Nonce     string           `json:"nonce"`
	BodyHash  string           `json:"bodyHash"`
	Signature string           `json:"signature"`
	PoWProof  *powProofPayload `json:"powProof,omitempty"`
}

type powProofPayload struct {
	Challenge  string `json:"challenge"`
	Nonce      string `json:"nonce"`
	Hash       string `json:"hash"`
	Difficulty int    `json:"difficulty"`
}

func (p browserProofPayload) toDomain() domainsecurity.RequestProof {
	result := domainsecurity.RequestProof{
		KeyID:     p.KeyID,
		SessionID: p.SessionID,
		Timestamp: p.Timestamp,
		Nonce:     p.Nonce,
		BodyHash:  p.BodyHash,
		Signature: p.Signature,
	}
	if p.PoWProof != nil {
		result.PoWProof = &domainsecurity.PoWProof{
			Challenge:  p.PoWProof.Challenge,
			Nonce:      p.PoWProof.Nonce,
			Hash:       p.PoWProof.Hash,
			Difficulty: p.PoWProof.Difficulty,
		}
	}
	return result
}

func browserProofAction(method string, requestPath string) (string, bool) {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	for _, route := range protectedBrowserProofRoutes {
		if route.method == normalizedMethod && route.pattern.MatchString(requestPath) {
			return route.action, true
		}
	}
	return "", false
}

func readAndRestoreRequestBody(c *gin.Context) ([]byte, error) {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}
