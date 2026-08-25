package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/security"
	"github.com/gin-gonic/gin"
)

func TestBrowserProofMiddlewareRejectsProtectedEndpointWithoutProof(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyUserID, uint(7))
		c.Set(ContextKeySessionID, "session-1")
		c.Next()
	})
	router.Use(BrowserProofMiddleware(&fakeBrowserProofVerifier{}))
	router.POST("/api/v1/conversations/:id/messages/stream", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/abc/messages/stream", bytes.NewBufferString(`{"content":"hello"}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"errorCode":"browser_proof.required"`) {
		t.Fatalf("expected stable browser proof error code, body=%s", recorder.Body.String())
	}
}

func TestBrowserProofActionIncludesVideoGeneration(t *testing.T) {
	action, ok := browserProofAction(http.MethodPost, "/api/v1/conversations/abc/media/videos/generations/stream")
	if !ok || action != "generate_video" {
		t.Fatalf("expected generate_video protected action, got action=%q ok=%v", action, ok)
	}
}

func TestBrowserProofMiddlewareVerifiesProtectedEndpointAndRestoresBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := &fakeBrowserProofVerifier{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyUserID, uint(7))
		c.Set(ContextKeySessionID, "session-1")
		c.Next()
	})
	router.Use(BrowserProofMiddleware(verifier))
	router.POST("/api/v1/conversations/:id/messages", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Fatalf("read restored body: %v", err)
		}
		c.String(http.StatusOK, string(body))
	})

	proofPayload, _ := json.Marshal(map[string]interface{}{
		"keyId":     "key-1",
		"sessionId": "session-1",
		"timestamp": int64(1700000000000),
		"nonce":     "nonce-1",
		"bodyHash":  "body-hash",
		"signature": "signature",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/abc/messages", bytes.NewBufferString(`{"content":"hello"}`))
	request.Header.Set("X-DEEIX-Proof", string(proofPayload))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != `{"content":"hello"}` {
		t.Fatalf("expected restored body, got %q", recorder.Body.String())
	}
	if string(verifier.lastInput.Body) != `{"content":"hello"}` {
		t.Fatalf("expected verifier body, got %q", string(verifier.lastInput.Body))
	}
	if verifier.lastInput.Action != "send_message" {
		t.Fatalf("expected send_message action, got %q", verifier.lastInput.Action)
	}
}

func TestBrowserProofMiddlewareSkipsUnprotectedEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(BrowserProofMiddleware(&fakeBrowserProofVerifier{err: appsecurity.ErrRequestProofMissing}))
	router.GET("/api/v1/models", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
}

type fakeBrowserProofVerifier struct {
	err       error
	lastInput BrowserProofVerifyInput
}

func (f *fakeBrowserProofVerifier) VerifyRequestProof(_ context.Context, input BrowserProofVerifyInput) error {
	f.lastInput = input
	return f.err
}
