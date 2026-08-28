package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	channelhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/channel"
	"github.com/gin-gonic/gin"
)

func TestVersionEndpointIsPublicAndUncached(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, err := NewEngine(config.NewRuntime(config.Config{AppName: "test", JWTSecret: "test-jwt-secret-value"}), nil, Modules{}, nil, nil)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store, no-cache, must-revalidate" {
		t.Fatalf("expected version no-store cache header, got %q", got)
	}
	if got := recorder.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("expected version pragma no-cache, got %q", got)
	}
	if got := recorder.Header().Get("Content-Security-Policy"); got != "" {
		t.Fatalf("expected API response without Content-Security-Policy, got %q", got)
	}
	if !strings.Contains(recorder.Body.String(), `"buildID"`) {
		t.Fatalf("expected version response to include buildID, got %q", recorder.Body.String())
	}
}

func TestNewEngineRegistersAllChannelPublicRoutesOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	channelModule := channelhttp.NewModule(channelhttp.NewHandler(nil))
	engine, err := NewEngine(
		config.NewRuntime(config.Config{AppName: "test", JWTSecret: "test-jwt-secret-value"}),
		nil,
		Modules{Channel: channelModule},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("create engine with channel module: %v", err)
	}

	expectedPaths := map[string]int{
		"/api/v1/public/models":              0,
		"/api/v1/llm/icon-assets/:public_id": 0,
	}
	for _, route := range engine.Routes() {
		if route.Method == http.MethodGet {
			if _, ok := expectedPaths[route.Path]; ok {
				expectedPaths[route.Path]++
			}
		}
	}
	for path, registrations := range expectedPaths {
		if registrations != 1 {
			t.Fatalf("route %s registrations = %d, want 1", path, registrations)
		}
	}
}

func TestFrontendStaticFallbackServesExportedPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "chat.html"), []byte("chat page"), 0o644); err != nil {
		t.Fatalf("write chat: %v", err)
	}

	engine := gin.New()
	registerFrontendStatic(engine, root, nil, nil, os.ReadFile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/chat?conversation_id=demo", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if strings.TrimSpace(recorder.Body.String()) != "chat page" {
		t.Fatalf("expected chat page, got %q", recorder.Body.String())
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("expected exported page no-cache, got %q", got)
	}
	if got := recorder.Header().Get("Content-Security-Policy"); got != "" {
		t.Fatalf("expected exported page without Content-Security-Policy, got %q", got)
	}
}

type fakeShareMetadataProvider struct {
	title       string
	description string
}

func (f fakeShareMetadataProvider) GetPublicShareMetadata(_ context.Context, _ string) (string, string, error) {
	return f.title, f.description, nil
}

func TestFrontendSharePathServesExportedSharePageWithOpenGraphMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	const shareHTML = `<!doctype html><html><head><title>DEEIX Chat</title><meta name="description" content="Default"></head><body>share app</body></html>`
	if err := os.WriteFile(filepath.Join(root, "share.html"), []byte(shareHTML), 0o644); err != nil {
		t.Fatalf("write share page: %v", err)
	}

	engine := gin.New()
	registerFrontendStatic(engine, root, nil, fakeShareMetadataProvider{
		title:       "Team plan",
		description: `Use "alpha" < beta & ship.`,
	}, os.ReadFile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "https://chat.example/share/abc123", nil)
	engine.ServeHTTP(recorder, request)

	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(body, "share app") {
		t.Fatalf("expected share app body, got %q", body)
	}
	if !strings.Contains(body, `<meta property="og:title" content="Team plan · DEEIX Chat">`) {
		t.Fatalf("expected injected og:title, got %q", body)
	}
	if !strings.Contains(body, `<meta name="description" content="Use &#34;alpha&#34; &lt; beta &amp; ship.">`) {
		t.Fatalf("expected escaped description, got %q", body)
	}
	if !strings.Contains(body, `<meta property="og:url" content="https://chat.example/share/abc123">`) {
		t.Fatalf("expected canonical og:url, got %q", body)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("expected share page no-cache, got %q", got)
	}
}

func TestFrontendShareQueryServesExportedSharePageWithCanonicalOpenGraphURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "share.html"), []byte(`<!doctype html><html><head></head><body>share app</body></html>`), 0o644); err != nil {
		t.Fatalf("write share page: %v", err)
	}

	engine := gin.New()
	registerFrontendStatic(engine, root, nil, fakeShareMetadataProvider{
		title:       "Legacy link",
		description: "Legacy description",
	}, os.ReadFile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "https://chat.example/share?conversation_id=legacy-id", nil)
	engine.ServeHTTP(recorder, request)

	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(body, `<link rel="canonical" href="https://chat.example/share/legacy-id">`) {
		t.Fatalf("expected canonical path for legacy query share link, got %q", body)
	}
}

func TestFrontendStaticFallbackUsesAcceptLanguageLocaleDirectory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	zhDir := filepath.Join(root, "zh-CN")
	enDir := filepath.Join(root, "en-US")
	if err := os.MkdirAll(zhDir, 0o755); err != nil {
		t.Fatalf("create zh dir: %v", err)
	}
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("create en dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(zhDir, "chat.html"), []byte("zh chat"), 0o644); err != nil {
		t.Fatalf("write zh chat: %v", err)
	}
	if err := os.WriteFile(filepath.Join(enDir, "chat.html"), []byte("en chat"), 0o644); err != nil {
		t.Fatalf("write en chat: %v", err)
	}

	engine := gin.New()
	registerFrontendStatic(engine, root, nil, nil, os.ReadFile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/chat", nil)
	request.Header.Set("Accept-Language", "en-US,en;q=0.8,zh-CN;q=0.5")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if strings.TrimSpace(recorder.Body.String()) != "en chat" {
		t.Fatalf("expected en chat page, got %q", recorder.Body.String())
	}
	if got := recorder.Header().Get("Vary"); !strings.Contains(got, "Accept-Language") {
		t.Fatalf("expected Vary to include Accept-Language, got %q", got)
	}
}

func TestFrontendStaticFallbackCookieLocaleBeatsAcceptLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	zhDir := filepath.Join(root, "zh-CN")
	enDir := filepath.Join(root, "en-US")
	if err := os.MkdirAll(zhDir, 0o755); err != nil {
		t.Fatalf("create zh dir: %v", err)
	}
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("create en dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(zhDir, "index.html"), []byte("zh index"), 0o644); err != nil {
		t.Fatalf("write zh index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(enDir, "index.html"), []byte("en index"), 0o644); err != nil {
		t.Fatalf("write en index: %v", err)
	}

	engine := gin.New()
	registerFrontendStatic(engine, root, nil, nil, os.ReadFile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "deeix_chat_locale", Value: "zh-CN"})
	request.Header.Set("Accept-Language", "en-US,en;q=0.8")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if strings.TrimSpace(recorder.Body.String()) != "zh index" {
		t.Fatalf("expected zh index page, got %q", recorder.Body.String())
	}
}

func TestFrontendStaticCachesNextExportData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "__next._tree.txt"), []byte("tree"), 0o644); err != nil {
		t.Fatalf("write next data: %v", err)
	}

	engine := gin.New()
	registerFrontendStatic(engine, root, nil, nil, os.ReadFile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/__next._tree.txt?conversation_id=demo&_rsc=abc", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=86400, stale-while-revalidate=604800" {
		t.Fatalf("expected next export data cache header, got %q", got)
	}
}

func TestFrontendStaticCachesImmutableBuildAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	chunkDir := filepath.Join(root, "_next", "static", "chunks")
	if err := os.MkdirAll(chunkDir, 0o755); err != nil {
		t.Fatalf("create chunk dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chunkDir, "app.js"), []byte("chunk"), 0o644); err != nil {
		t.Fatalf("write chunk: %v", err)
	}

	engine := gin.New()
	registerFrontendStatic(engine, root, nil, nil, os.ReadFile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/_next/static/chunks/app.js", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("expected immutable cache header, got %q", got)
	}
	if got := recorder.Header().Get("Content-Security-Policy"); got != "" {
		t.Fatalf("expected static asset without Content-Security-Policy, got %q", got)
	}
}

func TestFrontendStaticFallbackSkipsAPIPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	engine := gin.New()
	registerFrontendStatic(engine, root, nil, nil, os.ReadFile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "index") {
		t.Fatalf("api path should not serve frontend fallback: %q", recorder.Body.String())
	}
}

func TestSwaggerEnabledByEnvironment(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{env: "", want: false},
		{env: "dev", want: true},
		{env: " DEV ", want: true},
		{env: "development", want: true},
		{env: "staging", want: false},
		{env: "prod", want: false},
		{env: "production", want: false},
		{env: " PROD ", want: false},
	}

	for _, tt := range tests {
		if got := swaggerEnabled(tt.env); got != tt.want {
			t.Fatalf("swaggerEnabled(%q) = %v, want %v", tt.env, got, tt.want)
		}
	}
}
