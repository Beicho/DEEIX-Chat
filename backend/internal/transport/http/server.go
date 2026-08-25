package httpx

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/buildinfo"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	adminhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/admin"
	alertinghttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/alerting"
	announcementhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/announcement"
	authhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/auth"
	billinghttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/billing"
	channelhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/channel"
	contentmoderationhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/contentmoderation"
	conversationhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/conversation"
	knowledgebasehttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/knowledgebase"
	mcphttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/mcp"
	memoryhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/memory"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	notificationhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/notification"
	promptpresethttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/promptpreset"
	securityhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/security"
	settingshttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/settings"
	skillhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/skill"
	statushttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/status"
	userhttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/user"
	usersettingshttp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/usersettings"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

// HealthCheck 表示单个健康检查项的结果。
type HealthCheck struct {
	Name   string
	Status string
}

// HealthChecker 封装服务健康检查能力。
type HealthChecker interface {
	// CheckHealth 执行所有健康检查，返回检查结果列表。
	// 当所有检查均通过时 healthy 为 true。
	CheckHealth(ctx context.Context) (checks []HealthCheck, healthy bool)
}

// Modules 聚合可注册的业务模块。
type Modules struct {
	Auth              *authhttp.Module
	AuthService       middleware.SessionValidator
	Channel           *channelhttp.Module
	Conversation      *conversationhttp.Module
	MCP               *mcphttp.Module
	Memory            *memoryhttp.Module
	Billing           *billinghttp.Module
	Admin             *adminhttp.Module
	ContentModeration *contentmoderationhttp.Module
	Announcement      *announcementhttp.Module
	PromptPreset      *promptpresethttp.Module
	Skill             *skillhttp.Module
	KnowledgeBase     *knowledgebasehttp.Module
	Settings          *settingshttp.Module
	User              *userhttp.Module
	UserSettings      *usersettingshttp.Module
	StartupLog        func(*zap.Logger)
}

// NewEngine 创建并注册 API 路由。
func NewEngine(cfg *config.Runtime, log *zap.Logger, modules Modules, hc HealthChecker, limiter middleware.RateLimiter) (*gin.Engine, error) {
	snapshot := cfg.Snapshot()
	if snapshot.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.MaxMultipartMemory = 8 << 20
	if err := engine.SetTrustedProxies(snapshot.TrustedProxyList()); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}
	if err := middleware.ConfigureTrustedProxyHeaders(snapshot.TrustedProxyList()); err != nil {
		return nil, fmt.Errorf("configure trusted proxy headers: %w", err)
	}
	engine.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if log != nil {
			log.Error("http_panic_recovered", zap.Any("error", recovered), zap.ByteString("stack", debug.Stack()))
		}
		response.ErrorWithCode(c, http.StatusInternalServerError, response.CodeInternal, "internal server error")
		c.Abort()
	}))
	engine.Use(otelgin.Middleware(snapshot.AppName, otelgin.WithFilter(func(req *http.Request) bool {
		return req.URL.Path != "/healthz"
	})))
	engine.Use(middleware.RequestID())
	engine.Use(middleware.AccessLog(log))
	engine.Use(middleware.SecurityHeaders())
	engine.Use(middleware.CORS(snapshot.CORSAllowOrigin))

	engine.GET("/healthz", func(c *gin.Context) {
		info := buildinfo.Snapshot()
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": info.Version})
	})
	engine.GET("/readyz", readyzHandler(hc))
	if swaggerEnabled(snapshot.Env) {
		engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := engine.Group("/api/v1")
	api.GET("/version", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.JSON(http.StatusOK, buildinfo.Snapshot())
	})
	if modules.Auth != nil || modules.Settings != nil || modules.Billing != nil || modules.Conversation != nil || modules.User != nil || modules.Channel != nil {
		publicAuth := api.Group("")
		publicAuth.Use(middleware.PublicAuthRateLimit(limiter, cfg))
		if modules.Auth != nil {
			modules.Auth.RegisterPublicRoutes(publicAuth)
		}
		if modules.User != nil {
			modules.User.RegisterPublicRoutes(publicAuth)
		}
		if modules.Channel != nil {
			modules.Channel.RegisterPublicRoutes(publicAuth)
		}
		if modules.Conversation != nil {
			modules.Conversation.RegisterPublicRoutes(publicAuth)
		}
		if modules.Settings != nil {
			modules.Settings.RegisterPublicRoutes(publicAuth)
		}
		if modules.Billing != nil {
			modules.Billing.RegisterPublicRoutes(publicAuth)
		}
		if modules.Channel != nil {
			modules.Channel.RegisterPublicRoutes(publicAuth)
		}
	}

	authRequired := api.Group("")
	authRequired.Use(middleware.AuthMiddleware(snapshot.JWTSecret, modules.AuthService))
	if snapshot.BrowserProofEnabled && snapshot.RequestSigningEnabled && modules.BrowserProof != nil {
		authRequired.Use(middleware.BrowserProofMiddleware(modules.BrowserProof))
	}
	if modules.Fingerprint != nil {
		authRequired.Use(middleware.FingerprintMiddleware(modules.Fingerprint))
	}
	authRequired.Use(middleware.RateLimit(limiter, cfg))
	if slotStore, ok := limiter.(middleware.GenerationSlotStore); ok && slotStore != nil {
		authRequired.Use(middleware.GenerationConcurrencyLimit(slotStore, cfg))
	}

	if modules.Auth != nil {
		modules.Auth.RegisterProtectedRoutes(authRequired)
	}
	if modules.Security != nil {
		modules.Security.RegisterRoutes(authRequired)
	}
	if modules.Conversation != nil {
		modules.Conversation.RegisterRoutes(authRequired)
	}
	if modules.Channel != nil {
		modules.Channel.RegisterRoutes(authRequired)
	}
	if modules.Memory != nil {
		modules.Memory.RegisterRoutes(authRequired)
	}
	if modules.MCP != nil {
		modules.MCP.RegisterRoutes(authRequired)
	}
	if modules.Billing != nil {
		modules.Billing.RegisterRoutes(authRequired)
	}
	if modules.Announcement != nil {
		modules.Announcement.RegisterRoutes(authRequired)
	}
	if modules.Notification != nil {
		modules.Notification.RegisterRoutes(authRequired)
	}
	if modules.Collaboration != nil {
		modules.Collaboration.RegisterRoutes(authRequired)
	}
	if modules.PromptPreset != nil {
		modules.PromptPreset.RegisterRoutes(authRequired)
	}
	if modules.Skill != nil {
		modules.Skill.RegisterRoutes(authRequired)
	}
	if modules.KnowledgeBase != nil {
		modules.KnowledgeBase.RegisterRoutes(authRequired)
	}
	if modules.UserSettings != nil {
		modules.UserSettings.RegisterRoutes(authRequired)
	}
	if modules.Settings != nil {
		modules.Settings.RegisterRoutes(authRequired)
	}
	if modules.User != nil {
		modules.User.RegisterRoutes(authRequired)
	}
	if modules.Admin != nil || modules.Auth != nil || modules.Billing != nil || modules.Channel != nil || modules.MCP != nil || modules.Settings != nil || modules.Announcement != nil || modules.PromptPreset != nil || modules.Skill != nil || modules.KnowledgeBase != nil || modules.ContentModeration != nil {
		adminGroup := authRequired.Group("/admin")
		adminGroup.Use(middleware.AdminOnly())
		if modules.Auth != nil {
			modules.Auth.RegisterAdminRoutes(adminGroup)
		}
		if modules.Admin != nil {
			modules.Admin.RegisterRoutes(adminGroup)
		}
		if modules.ContentModeration != nil {
			modules.ContentModeration.RegisterRoutes(adminGroup)
		}
		if modules.Billing != nil {
			modules.Billing.RegisterAdminRoutes(adminGroup)
		}
		if modules.Channel != nil {
			modules.Channel.RegisterAdminRoutes(adminGroup)
		}
		if modules.Conversation != nil {
			modules.Conversation.RegisterAdminRoutes(adminGroup)
		}
		if modules.MCP != nil {
			modules.MCP.RegisterAdminRoutes(adminGroup)
		}
		if modules.Settings != nil {
			modules.Settings.RegisterAdminRoutes(adminGroup)
		}
		if modules.Security != nil {
			modules.Security.RegisterAdminRoutes(adminGroup)
		}
		if modules.Announcement != nil {
			modules.Announcement.RegisterAdminRoutes(adminGroup)
		}
		if modules.Alerting != nil {
			modules.Alerting.RegisterAdminRoutes(adminGroup)
		}
		if modules.PromptPreset != nil {
			modules.PromptPreset.RegisterAdminRoutes(adminGroup)
		}
		if modules.Skill != nil {
			modules.Skill.RegisterAdminRoutes(adminGroup)
		}
		if modules.KnowledgeBase != nil {
			modules.KnowledgeBase.RegisterAdminRoutes(adminGroup)
		}
	}

	if modules.StartupLog != nil {
		modules.StartupLog(log)
	}
	if modules.Settings != nil {
		modules.Settings.RegisterFrontendRoutes(engine)
	}
	registerFrontendStatic(engine, snapshot.FrontendDistDir, log)

	return engine, nil
}

func registerFrontendStatic(engine *gin.Engine, distDir string, log *zap.Logger, shareMetadata frontendShareMetadataProvider) {
	root := strings.TrimSpace(distDir)
	if root == "" {
		return
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		if log != nil {
			log.Warn("frontend_static_path_invalid", zap.String("path", root), zap.Error(err))
		}
		return
	}

	info, err := os.Stat(absoluteRoot)
	if err != nil || !info.IsDir() {
		if log != nil {
			log.Warn("frontend_static_disabled", zap.String("path", absoluteRoot), zap.Error(err))
		}
		return
	}

	if log != nil {
		log.Info("frontend_static_enabled", zap.String("path", absoluteRoot))
	}

	engine.NoRoute(func(c *gin.Context) {
		requestPath := cleanFrontendPath(c.Request.URL.Path)
		if isBackendOnlyPath(requestPath) {
			response.ErrorWithCode(c, http.StatusNotFound, response.CodeResourceNotFound, "not found")
			return
		}

		frontendRoots := resolveFrontendRoots(absoluteRoot, c.Request)
		if filePath, ok := resolveFrontendStaticFile(frontendRoots, requestPath); ok {
			applyFrontendCacheHeaders(c, requestPath)
			c.File(filePath)
			return
		}

		if shareID := frontendShareIDFromRequest(c.Request, requestPath); shareID != "" {
			if filePath, ok := resolveFrontendSharePageFile(frontendRoots); ok {
				c.Header("Cache-Control", "no-cache")
				applyFrontendLocaleVary(c)
				serveFrontendSharePage(c, filePath, shareID, shareMetadata)
				return
			}
		}

		if filePath, ok := resolveFrontendPageFile(frontendRoots, requestPath); ok {
			c.Header("Cache-Control", "no-cache")
			applyFrontendLocaleVary(c)
			c.File(filePath)
			return
		}

		for _, root := range frontendRoots {
			notFoundPath := filepath.Join(root, "404.html")
			if isRegularFile(notFoundPath) {
				c.Status(http.StatusNotFound)
				applyFrontendLocaleVary(c)
				c.File(notFoundPath)
				return
			}
		}

		response.ErrorWithCode(c, http.StatusNotFound, response.CodeResourceNotFound, "not found")
	})
}

func frontendShareIDFromRequest(request *http.Request, requestPath string) string {
	if request == nil {
		return ""
	}
	if requestPath == "/share" {
		return strings.TrimSpace(request.URL.Query().Get("conversation_id"))
	}
	if !strings.HasPrefix(requestPath, "/share/") {
		return ""
	}
	raw := strings.TrimPrefix(requestPath, "/share/")
	if raw == "" || strings.Contains(raw, "/") {
		return ""
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return strings.TrimSpace(decoded)
}

func resolveFrontendSharePageFile(roots []string) (string, bool) {
	for _, root := range roots {
		candidates := []string{
			filepath.Join(root, "share.html"),
			filepath.Join(root, "share", "index.html"),
		}
		for _, candidate := range candidates {
			if strings.HasPrefix(candidate, root) && isRegularFile(candidate) {
				return candidate, true
			}
		}
	}
	return "", false
}

func serveFrontendSharePage(c *gin.Context, filePath string, shareID string, provider frontendShareMetadataProvider) {
	body, err := os.ReadFile(filePath)
	if err != nil {
		c.File(filePath)
		return
	}
	pageHTML := string(body)
	if provider != nil {
		title, description, metadataErr := provider.GetPublicShareMetadata(c.Request.Context(), shareID)
		if metadataErr == nil {
			pageHTML = injectFrontendShareMetadata(pageHTML, frontendShareMetadata{
				Title:       title,
				Description: description,
				Canonical:   frontendShareCanonicalURL(c.Request, shareID),
				Image:       frontendAbsoluteURL(c.Request, "/DEEIX-Chat.jpg"),
			})
		}
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(pageHTML))
}

type frontendShareMetadata struct {
	Title       string
	Description string
	Canonical   string
	Image       string
}

func injectFrontendShareMetadata(pageHTML string, metadata frontendShareMetadata) string {
	title := strings.TrimSpace(metadata.Title)
	if title == "" {
		title = "Shared conversation"
	}
	if !strings.Contains(title, "DEEIX Chat") {
		title = title + " · DEEIX Chat"
	}
	description := strings.TrimSpace(metadata.Description)
	if description == "" {
		description = "Shared conversation on DEEIX Chat."
	}
	tags := []string{
		`<title>` + html.EscapeString(title) + `</title>`,
		`<meta name="description" content="` + html.EscapeString(description) + `">`,
		`<meta property="og:type" content="article">`,
		`<meta property="og:site_name" content="DEEIX Chat">`,
		`<meta property="og:title" content="` + html.EscapeString(title) + `">`,
		`<meta property="og:description" content="` + html.EscapeString(description) + `">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<meta name="twitter:title" content="` + html.EscapeString(title) + `">`,
		`<meta name="twitter:description" content="` + html.EscapeString(description) + `">`,
	}
	if canonical := strings.TrimSpace(metadata.Canonical); canonical != "" {
		escaped := html.EscapeString(canonical)
		tags = append(tags,
			`<link rel="canonical" href="`+escaped+`">`,
			`<meta property="og:url" content="`+escaped+`">`,
		)
	}
	if image := strings.TrimSpace(metadata.Image); image != "" {
		escaped := html.EscapeString(image)
		tags = append(tags,
			`<meta property="og:image" content="`+escaped+`">`,
			`<meta name="twitter:image" content="`+escaped+`">`,
		)
	}

	injection := strings.Join(tags, "\n")
	if strings.Contains(pageHTML, "<head>") {
		return strings.Replace(pageHTML, "<head>", "<head>\n"+injection+"\n", 1)
	}
	if strings.Contains(pageHTML, "<html") {
		head := "<head>\n" + injection + "\n</head>"
		index := strings.Index(pageHTML, ">")
		if index >= 0 {
			return pageHTML[:index+1] + head + pageHTML[index+1:]
		}
	}
	return injection + "\n" + pageHTML
}

func frontendShareCanonicalURL(request *http.Request, shareID string) string {
	return frontendAbsoluteURL(request, "/share/"+url.PathEscape(strings.TrimSpace(shareID)))
}

func frontendAbsoluteURL(request *http.Request, requestPath string) string {
	if request == nil {
		return requestPath
	}
	host := firstHeaderValue(request.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = request.Host
	}
	if host == "" {
		return requestPath
	}
	scheme := firstHeaderValue(request.Header.Get("X-Forwarded-Proto"))
	if scheme == "" && request.URL != nil {
		scheme = request.URL.Scheme
	}
	if scheme == "" {
		if request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + host + requestPath
}

func firstHeaderValue(value string) string {
	if value == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(value, ",")[0])
}

func swaggerEnabled(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "dev", "development":
		return true
	default:
		return false
	}
}

func cleanFrontendPath(rawPath string) string {
	if rawPath == "" || rawPath == "/" {
		return "/"
	}
	return path.Clean("/" + strings.TrimPrefix(rawPath, "/"))
}

func isBackendOnlyPath(requestPath string) bool {
	return requestPath == "/api" ||
		strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/swagger" ||
		strings.HasPrefix(requestPath, "/swagger/") ||
		requestPath == "/healthz" ||
		requestPath == "/readyz"
}

func resolveFrontendRoots(root string, request *http.Request) []string {
	locale := resolveFrontendLocale(request)
	roots := []string{}
	if locale != "" {
		localeRoot := filepath.Join(root, locale)
		if info, err := os.Stat(localeRoot); err == nil && info.IsDir() {
			roots = append(roots, localeRoot)
		}
	}
	if locale != "zh-CN" {
		defaultRoot := filepath.Join(root, "zh-CN")
		if info, err := os.Stat(defaultRoot); err == nil && info.IsDir() {
			roots = append(roots, defaultRoot)
		}
	}
	roots = append(roots, root)
	return roots
}

func resolveFrontendLocale(request *http.Request) string {
	if request != nil {
		if cookie, err := request.Cookie("deeix_chat_locale"); err == nil {
			if locale := normalizeFrontendLocale(cookie.Value); locale != "" {
				return locale
			}
		}
		for _, part := range strings.Split(request.Header.Get("Accept-Language"), ",") {
			language := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
			if locale := normalizeFrontendLocale(language); locale != "" {
				return locale
			}
		}
	}
	return "zh-CN"
}

func normalizeFrontendLocale(value string) string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))
	switch {
	case normalized == "zh" || strings.HasPrefix(normalized, "zh-"):
		return "zh-CN"
	case normalized == "en" || strings.HasPrefix(normalized, "en-"):
		return "en-US"
	default:
		return ""
	}
}

func applyFrontendLocaleVary(c *gin.Context) {
	c.Header("Vary", appendVaryHeader(c.Writer.Header().Get("Vary"), "Cookie", "Accept-Language"))
}

func appendVaryHeader(current string, values ...string) string {
	seen := make(map[string]bool)
	parts := make([]string, 0, len(values)+1)
	for _, item := range strings.Split(current, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		parts = append(parts, trimmed)
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, ", ")
}

func resolveFrontendStaticFile(roots []string, requestPath string) (string, bool) {
	if requestPath == "/" {
		return "", false
	}
	for _, root := range roots {
		candidate := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(requestPath, "/")))
		if !strings.HasPrefix(candidate, root) {
			continue
		}
		if isRegularFile(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func resolveFrontendPageFile(roots []string, requestPath string) (string, bool) {
	for _, root := range roots {
		candidates := []string{filepath.Join(root, "index.html")}
		if requestPath != "/" {
			cleanPath := filepath.FromSlash(strings.TrimPrefix(requestPath, "/"))
			candidates = []string{
				filepath.Join(root, cleanPath+".html"),
				filepath.Join(root, cleanPath, "index.html"),
				filepath.Join(root, "index.html"),
			}
		}

		for _, candidate := range candidates {
			if strings.HasPrefix(candidate, root) && isRegularFile(candidate) {
				return candidate, true
			}
		}
	}
	return "", false
}

func isRegularFile(filePath string) bool {
	info, err := os.Stat(filePath)
	return err == nil && !info.IsDir()
}

func applyFrontendCacheHeaders(c *gin.Context, requestPath string) {
	if isImmutableFrontendAsset(requestPath) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	if isVendorIconAsset(requestPath) {
		c.Header("Cache-Control", "public, max-age=86400, stale-while-revalidate=604800")
		return
	}
	if isNextExportDataAsset(requestPath) {
		c.Header("Cache-Control", "public, max-age=86400, stale-while-revalidate=604800")
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
}

func isImmutableFrontendAsset(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/_next/static/") ||
		strings.HasPrefix(requestPath, "/fonts/")
}

func isVendorIconAsset(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/vendor/lobehub-icons/")
}

func isNextExportDataAsset(requestPath string) bool {
	fileName := path.Base(requestPath)
	return strings.HasPrefix(fileName, "__next.") && strings.EqualFold(path.Ext(fileName), ".txt")
}

func readyzHandler(hc HealthChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var healthy bool
		checksMap := gin.H{}

		if hc != nil {
			results, ok := hc.CheckHealth(ctx)
			healthy = ok
			for _, r := range results {
				checksMap[r.Name] = r.Status
			}
		} else {
			healthy = true
		}

		status := http.StatusOK
		if !healthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{
			"status": map[bool]string{true: "ok", false: "degraded"}[healthy],
			"checks": checksMap,
		})
	}
}
