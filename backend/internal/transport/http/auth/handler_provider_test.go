package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appauth "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/auth"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/identityprovider"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/pkg/secretbox"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

func TestCompleteProviderLoginReturnsSuspensionReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataKey := "test-data-key"
	jwtSecret := "test-jwt-secret"
	clientSecret, err := secretbox.EncryptString(dataKey, "client-secret")
	if err != nil {
		t.Fatalf("encrypt client secret: %v", err)
	}
	providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			_ = r.ParseForm()
			if r.Form.Get("code") != "oauth-code" {
				t.Fatalf("unexpected authorization code %q", r.Form.Get("code"))
			}
			_, _ = w.Write([]byte(`{"access_token":"provider-token","token_type":"Bearer"}`))
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer provider-token" {
				t.Fatalf("unexpected authorization header %q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"sub":"sub-1","email":"bound@example.com","email_verified":true,"name":"Bound User"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer providerServer.Close()

	suspendedAt := time.Now().UTC().Truncate(time.Second)
	repo := &providerCallbackRepo{
		provider: &user.IdentityProvider{
			ID:                  10,
			Type:                user.IdentityProviderTypeOAuth2,
			Name:                "Acme SSO",
			Slug:                "acme",
			LoginEnabled:        true,
			RegistrationEnabled: true,
			ClientID:            "client-id",
			ClientSecret:        clientSecret,
			AuthURL:             providerServer.URL + "/auth",
			TokenURL:            providerServer.URL + "/token",
			UserInfoURL:         providerServer.URL + "/userinfo",
			SubjectField:        "sub",
			EmailField:          "email",
			EmailVerifiedField:  "email_verified",
			NameField:           "name",
		},
		identity: user.UserIdentity{ID: 7, UserID: 42, ProviderID: 10, ProviderSubject: "sub-1"},
		userItem: &user.User{
			ID:               42,
			Status:           user.StatusSuspended,
			SuspensionReason: "疑似同人多账号",
			SuspensionDetail: "设备指纹重复",
			SuspendedAt:      &suspendedAt,
		},
	}
	authConfig := config.Config{
		JWTSecret:              jwtSecret,
		DataEncryptionKey:      dataKey,
		ThirdPartyLoginEnabled: true,
	}
	service := appauth.NewServiceWithRuntime(
		config.NewRuntime(authConfig),
		repo,
		nil,
		identityprovider.New(authConfig.StrictOutboundPolicy()),
	)
	handler := NewHandler(service)
	router := gin.New()
	router.Use(middleware.RequestID())
	router.POST("/auth/providers/:slug/callback", handler.CompleteProviderLogin)

	codeVerifier := strings.Repeat("a", 43)
	redirectURI := "http://localhost/auth/callback?provider=acme"
	state := signedProviderState(t, jwtSecret, map[string]any{
		"provider":      "acme",
		"redirectURI":   redirectURI,
		"next":          "/chat",
		"intent":        "login",
		"codeChallenge": providerCodeChallengeForTest(codeVerifier),
		"expiresAt":     time.Now().Add(time.Minute).Unix(),
	})
	body := `{"code":"oauth-code","state":` + strconvQuote(state) + `,"redirectURI":` + strconvQuote(redirectURI) + `,"codeVerifier":` + strconvQuote(codeVerifier) + `,"intent":"login"}`
	request := httptest.NewRequest(http.MethodPost, "/auth/providers/acme/callback", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		ErrorCode string         `json:"errorCode"`
		Details   map[string]any `json:"details"`
	}
	if err = json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ErrorCode != "auth.account_suspended" {
		t.Fatalf("expected auth.account_suspended, got %q", payload.ErrorCode)
	}
	if payload.Details["reason"] != "疑似同人多账号" {
		t.Fatalf("expected reason detail, got %#v", payload.Details)
	}
	if payload.Details["detail"] != "设备指纹重复" {
		t.Fatalf("expected suspension detail, got %#v", payload.Details)
	}
	if repo.updateIdentityLoginCount != 0 {
		t.Fatalf("expected identity login not to be updated, got %d", repo.updateIdentityLoginCount)
	}
}

type providerCallbackRepo struct {
	repository.AuthRepository

	provider                 *user.IdentityProvider
	identity                 user.UserIdentity
	userItem                 *user.User
	updateIdentityLoginCount int
}

func (r *providerCallbackRepo) GetIdentityProviderBySlug(ctx context.Context, slug string) (*user.IdentityProvider, error) {
	if r.provider == nil || r.provider.Slug != slug {
		return nil, repository.ErrNotFound
	}
	return r.provider, nil
}

func (r *providerCallbackRepo) GetUserIdentityByProviderSubject(ctx context.Context, providerID uint, subject string) (*user.UserIdentity, error) {
	if r.identity.ProviderID != providerID || r.identity.ProviderSubject != subject {
		return nil, repository.ErrNotFound
	}
	return &r.identity, nil
}

func (r *providerCallbackRepo) GetByID(ctx context.Context, userID uint) (*user.User, error) {
	if r.userItem == nil || r.userItem.ID != userID {
		return nil, repository.ErrNotFound
	}
	return r.userItem, nil
}

func (r *providerCallbackRepo) UpdateUserIdentityLogin(ctx context.Context, identityID uint, profileJSON string, providerDisplayName string, email string, emailVerified bool) error {
	r.updateIdentityLoginCount++
	return nil
}

func (r *providerCallbackRepo) RecordAuthEvent(ctx context.Context, userID uint, requestID string, eventType string, result string, reason string, clientIP string, userAgent string, detailJSON string) error {
	return nil
}

func signedProviderState(t *testing.T, secret string, payload map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal provider state: %v", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encodedPayload))
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func providerCodeChallengeForTest(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func strconvQuote(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
