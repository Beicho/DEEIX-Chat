package mcp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	systemeventapp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/systemevent"
	domainmcp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/mcp"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	inframcp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/mcp"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/pkg/secretbox"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

var (
	ErrInvalidServerName           = errors.New("invalid mcp server name")
	ErrInvalidServerBaseURL        = errors.New("invalid mcp server base url")
	ErrInvalidServerStatus         = errors.New("invalid mcp server status")
	ErrInvalidServerHeaders        = errors.New("invalid mcp server headers json")
	ErrInvalidToolStatus           = errors.New("invalid mcp tool status")
	ErrInvalidToolName             = errors.New("invalid mcp tool display name")
	ErrInvalidToolDesc             = errors.New("invalid mcp tool description")
	ErrInvalidToolAttachmentConfig = errors.New("invalid mcp tool attachment configuration")
	ErrInvalidToolSelection        = errors.New("invalid mcp tool selection")
	ErrServerNotFound              = errors.New("mcp server not found")
	ErrMCPClientUnavailable        = errors.New("mcp client unavailable")
)

const (
	defaultMCPServerToolListTimeoutMS = 30000
	maxServerTimeoutSeconds           = 300
)

type Service struct {
	cfg               *config.Runtime
	repo              repository.MCPRepository
	client            *inframcp.Client
	systemEventWriter systemEventWriter
}

type ReorderServerInput struct {
	ServerID uint
	ToolIDs  []uint
}

type systemEventWriter interface {
	Write(ctx context.Context, input systemeventapp.WriteInput)
}

type ServerInput struct {
	OwnerUserID       uint
	Name              string
	BaseURL           string
	AuthToken         string
	HeadersJSON       string
	Status            string
	TimeoutSeconds    int
	OAuthClientID     string
	OAuthClientSecret string
	OAuthAuthURL      string
	OAuthTokenURL     string
	OAuthScopes       string
	OAuthAccessToken  string
	OAuthRefreshToken string
}

type ToolInput struct {
	DisplayName              *string
	Description              *string
	AttachmentInputMode      *string
	AttachmentArgument       *string
	AttachmentEncoding       *string
	AttachmentPromptArgument *string
	Status                   *string
}

// SyncServerToolsInput 描述一次 MCP 工具同步请求。
type SyncServerToolsInput struct {
	ServerID                    uint
	RequestID                   string
	OverwriteCustomizedMetadata bool
}

// ToolPreferenceInput 描述用户工具选择偏好。
type ToolPreferenceInput struct {
	ConversationPublicID string
	SelectedToolIDs      []uint
	ConfirmedToolIDs     []uint
	WebSearchEnabled     bool
	CodeSandboxEnabled   bool
	ResearchMaxLLMCalls  int
	ResearchMaxToolCalls int
}

// ConnectionTestResult 是连接测试的用户友好结果。
type ConnectionTestResult struct {
	OK        bool
	ErrorCode string
	Message   string
	ToolCount int
}

// OAuthStartResult 返回远程连接器授权入口。
type OAuthStartResult struct {
	AuthorizationURL string
	State            string
}

// VoiceConfigResult 是语音接口的公开配置摘要。
type VoiceConfigResult struct {
	ASREnabled  bool
	ASRProvider string
	ASRModel    string
	TTSEnabled  bool
	TTSProvider string
	TTSModel    string
	TTSVoice    string
}

// NewServiceWithRuntime 创建 MCP 应用服务。
func NewServiceWithRuntime(cfg *config.Runtime, repo repository.MCPRepository, client *inframcp.Client) *Service {
	return &Service{cfg: cfg, repo: repo, client: client}
}

// SetSystemEventWriter 注入系统事件写入器。
func (s *Service) SetSystemEventWriter(writer systemEventWriter) {
	s.systemEventWriter = writer
}

func (s *Service) ListServers(ctx context.Context) ([]domainmcp.Server, error) {
	return s.repo.ListServers(ctx)
}

func (s *Service) ListUserServers(ctx context.Context, userID uint) ([]domainmcp.Server, error) {
	return s.repo.ListServersForUser(ctx, userID, false)
}

func (s *Service) GetServer(ctx context.Context, serverID uint) (*domainmcp.Server, error) {
	return s.repo.GetServer(ctx, serverID)
}

func (s *Service) GetUserServer(ctx context.Context, userID uint, serverID uint) (*domainmcp.Server, error) {
	return s.repo.GetServerForUser(ctx, serverID, userID, false)
}

func (s *Service) CreateServer(ctx context.Context, input ServerInput) (*domainmcp.Server, error) {
	normalized, err := s.normalizeServerInput(input, true)
	if err != nil {
		return nil, err
	}
	tokenEnc, err := s.encryptToken(normalized.AuthToken)
	if err != nil {
		return nil, err
	}
	oauthClientSecretEnc, err := s.encryptOptionalToken(normalized.OAuthClientSecret)
	if err != nil {
		return nil, err
	}
	oauthAccessTokenEnc, err := s.encryptOptionalToken(normalized.OAuthAccessToken)
	if err != nil {
		return nil, err
	}
	oauthRefreshTokenEnc, err := s.encryptOptionalToken(normalized.OAuthRefreshToken)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateServer(ctx, repository.CreateMCPServerInput{
		OwnerUserID:          normalized.OwnerUserID,
		Name:                 normalized.Name,
		BaseURL:              normalized.BaseURL,
		AuthTokenEnc:         tokenEnc,
		HeadersJSON:          normalized.HeadersJSON,
		Status:               normalized.Status,
		TimeoutSeconds:       normalized.TimeoutSeconds,
		OAuthClientID:        normalized.OAuthClientID,
		OAuthClientSecretEnc: oauthClientSecretEnc,
		OAuthAuthURL:         normalized.OAuthAuthURL,
		OAuthTokenURL:        normalized.OAuthTokenURL,
		OAuthScopes:          normalized.OAuthScopes,
		OAuthAccessTokenEnc:  oauthAccessTokenEnc,
		OAuthRefreshTokenEnc: oauthRefreshTokenEnc,
	})
}

func (s *Service) CreateUserServer(ctx context.Context, userID uint, input ServerInput) (*domainmcp.Server, error) {
	input.OwnerUserID = userID
	return s.CreateServer(ctx, input)
}

func (s *Service) UpdateServer(ctx context.Context, serverID uint, input ServerInput) (*domainmcp.Server, error) {
	normalized, err := s.normalizeServerInput(input, false)
	if err != nil {
		return nil, err
	}
	update := repository.UpdateMCPServerInput{
		Name:           &normalized.Name,
		BaseURL:        &normalized.BaseURL,
		HeadersJSON:    &normalized.HeadersJSON,
		Status:         &normalized.Status,
		TimeoutSeconds: &normalized.TimeoutSeconds,
		OAuthClientID:  &normalized.OAuthClientID,
		OAuthAuthURL:   &normalized.OAuthAuthURL,
		OAuthTokenURL:  &normalized.OAuthTokenURL,
		OAuthScopes:    &normalized.OAuthScopes,
	}
	if normalized.AuthToken != "" {
		tokenEnc, encryptErr := s.encryptToken(normalized.AuthToken)
		if encryptErr != nil {
			return nil, encryptErr
		}
		update.AuthTokenEnc = &tokenEnc
	}
	if normalized.OAuthClientSecret != "" {
		value, encryptErr := s.encryptOptionalToken(normalized.OAuthClientSecret)
		if encryptErr != nil {
			return nil, encryptErr
		}
		update.OAuthClientSecretEnc = &value
	}
	if normalized.OAuthAccessToken != "" {
		value, encryptErr := s.encryptOptionalToken(normalized.OAuthAccessToken)
		if encryptErr != nil {
			return nil, encryptErr
		}
		update.OAuthAccessTokenEnc = &value
	}
	if normalized.OAuthRefreshToken != "" {
		value, encryptErr := s.encryptOptionalToken(normalized.OAuthRefreshToken)
		if encryptErr != nil {
			return nil, encryptErr
		}
		update.OAuthRefreshTokenEnc = &value
	}
	return s.repo.UpdateServer(ctx, serverID, update)
}

func (s *Service) UpdateUserServer(ctx context.Context, userID uint, serverID uint, input ServerInput) (*domainmcp.Server, error) {
	if _, err := s.repo.GetServerForUser(ctx, serverID, userID, false); err != nil {
		return nil, ErrServerNotFound
	}
	return s.UpdateServer(ctx, serverID, input)
}

func (s *Service) DeleteServer(ctx context.Context, serverID uint) error {
	return s.repo.DeleteServer(ctx, serverID)
}

func (s *Service) DeleteUserServer(ctx context.Context, userID uint, serverID uint) error {
	if _, err := s.repo.GetServerForUser(ctx, serverID, userID, false); err != nil {
		return ErrServerNotFound
	}
	return s.repo.DeleteServer(ctx, serverID)
}

func (s *Service) SyncServerTools(ctx context.Context, input SyncServerToolsInput) ([]domainmcp.Tool, error) {
	serverID := input.ServerID
	fail := func(err error) ([]domainmcp.Tool, error) {
		s.writeToolSyncEvent(ctx, input.RequestID, "error", "mcp.tools_sync_failed", serverID, "MCP 工具同步失败", map[string]interface{}{
			"server_id": serverID,
			"error":     err.Error(),
		})
		return nil, err
	}

	server, err := s.repo.GetServer(ctx, serverID)
	if err != nil {
		return fail(err)
	}
	if err = s.validateServerBaseURL(server.BaseURL); err != nil {
		return fail(err)
	}
	if s.client == nil {
		return fail(ErrMCPClientUnavailable)
	}
	token, err := s.decryptToken(server.AuthTokenEnc)
	if err != nil {
		return fail(err)
	}
	headers, err := parseHeadersJSON(server.HeadersJSON)
	if err != nil {
		return fail(err)
	}
	tools, err := s.client.ListTools(ctx, inframcp.CallConfig{
		BaseURL:   server.BaseURL,
		AuthToken: token,
		TimeoutMS: resolveServerTimeoutMS(server.TimeoutSeconds, s.cfg.Snapshot().MCPToolTimeoutSeconds, defaultMCPServerToolListTimeoutMS),
		Headers:   headers,
	})
	if err != nil {
		message := err.Error()
		_, _ = s.repo.UpdateServer(ctx, serverID, repository.UpdateMCPServerInput{LastError: &message})
		return fail(err)
	}
	existingTools, err := s.repo.ListTools(ctx, serverID, false)
	if err != nil {
		return fail(err)
	}
	existingToolsByName := make(map[string]domainmcp.Tool, len(existingTools))
	for _, tool := range existingTools {
		existingToolsByName[tool.Name] = tool
	}
	items := make([]domainmcp.Tool, 0, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		schema := strings.TrimSpace(string(tool.InputSchema))
		if schema == "" {
			schema = "{}"
		}
		displayName := strings.TrimSpace(tool.Title)
		if displayName == "" {
			displayName = name
		}
		item := domainmcp.Tool{
			ServerID:            serverID,
			Name:                name,
			DisplayName:         displayName,
			Description:         strings.TrimSpace(tool.Description),
			InputSchemaJSON:     schema,
			AttachmentInputMode: domainmcp.AttachmentInputModeNone,
			Status:              "active",
		}
		if existing, ok := existingToolsByName[name]; ok {
			preserveCompatibleToolAttachmentConfig(&item, existing)
		}
		items = append(items, item)
	}
	if err = s.repo.ReplaceServerTools(ctx, serverID, items, input.OverwriteCustomizedMetadata); err != nil {
		return fail(err)
	}
	result, err := s.repo.ListTools(ctx, serverID, false)
	if err != nil {
		return fail(err)
	}
	s.writeToolSyncEvent(ctx, input.RequestID, "info", "mcp.tools_synced", serverID, "MCP 工具已同步", map[string]interface{}{
		"server_id":                     serverID,
		"tool_count":                    len(result),
		"overwrite_customized_metadata": input.OverwriteCustomizedMetadata,
	})
	return result, nil
}

func preserveCompatibleToolAttachmentConfig(discovered *domainmcp.Tool, existing domainmcp.Tool) {
	if discovered == nil {
		return
	}
	config := toolAttachmentConfig{
		Mode:           strings.TrimSpace(existing.AttachmentInputMode),
		Argument:       strings.TrimSpace(existing.AttachmentArgument),
		Encoding:       strings.TrimSpace(existing.AttachmentEncoding),
		PromptArgument: strings.TrimSpace(existing.AttachmentPromptArgument),
	}
	if validateToolAttachmentConfig(config, discovered.InputSchemaJSON) != nil {
		return
	}
	discovered.AttachmentInputMode = config.Mode
	discovered.AttachmentArgument = config.Argument
	discovered.AttachmentEncoding = config.Encoding
	discovered.AttachmentPromptArgument = config.PromptArgument
}

func (s *Service) writeToolSyncEvent(ctx context.Context, requestID string, level string, event string, serverID uint, message string, detail interface{}) {
	if s.systemEventWriter == nil {
		return
	}
	s.systemEventWriter.Write(ctx, systemeventapp.WriteInput{
		RequestID:  strings.TrimSpace(requestID),
		Level:      level,
		Source:     "mcp",
		Event:      event,
		Resource:   "mcp_server",
		ResourceID: fmt.Sprintf("%d", serverID),
		Message:    message,
		Detail:     detail,
	})
}

func (s *Service) ListTools(ctx context.Context, serverID uint, onlyActive bool) ([]domainmcp.Tool, error) {
	return s.repo.ListTools(ctx, serverID, onlyActive)
}

func (s *Service) ListAvailableTools(ctx context.Context, userID uint) ([]domainmcp.Tool, error) {
	if !s.cfg.Snapshot().MCPEnable {
		return []domainmcp.Tool{}, nil
	}
	servers, err := s.repo.ListServersForUser(ctx, userID, true)
	if err != nil {
		return nil, err
	}
	result := make([]domainmcp.Tool, 0)
	for _, server := range servers {
		if server.Status != "active" {
			continue
		}
		tools, err := s.repo.ListTools(ctx, server.ID, true)
		if err != nil {
			return nil, err
		}
		for _, tool := range tools {
			tool.ServerName = server.Name
			result = append(result, tool)
		}
	}
	return result, nil
}

func (s *Service) UpdateTool(ctx context.Context, toolID uint, input ToolInput) (*domainmcp.Tool, error) {
	update, err := normalizeToolInput(input)
	if err != nil {
		return nil, err
	}
	if toolAttachmentConfigChanged(input) {
		tools, listErr := s.repo.ListToolsByIDs(ctx, []uint{toolID})
		if listErr != nil {
			return nil, listErr
		}
		if len(tools) != 1 || tools[0].ID != toolID {
			return nil, repository.ErrNotFound
		}
		config := mergedToolAttachmentConfig(tools[0], update)
		if validationErr := validateToolAttachmentConfig(config, tools[0].InputSchemaJSON); validationErr != nil {
			return nil, validationErr
		}
		if config.Mode == domainmcp.AttachmentInputModeNone {
			empty := ""
			update.AttachmentArgument = &empty
			update.AttachmentEncoding = &empty
			update.AttachmentPromptArgument = &empty
		}
	}
	return s.repo.UpdateTool(ctx, toolID, update)
}

func (s *Service) UpdateServerToolsStatus(ctx context.Context, serverID uint, toolIDs []uint, status string) ([]domainmcp.Tool, error) {
	normalized, err := normalizeToolStatus(status)
	if err != nil {
		return nil, err
	}
	if len(toolIDs) == 0 {
		return nil, ErrInvalidToolSelection
	}
	return s.repo.UpdateServerToolsStatus(ctx, serverID, toolIDs, normalized)
}

func (s *Service) GetToolPreference(ctx context.Context, userID uint, conversationPublicID string) (*domainmcp.ToolPreference, error) {
	item, err := s.repo.GetToolPreference(ctx, userID, normalizeConversationPublicID(conversationPublicID))
	if err == nil {
		return item, nil
	}
	return &domainmcp.ToolPreference{
		UserID:               userID,
		ConversationPublicID: normalizeConversationPublicID(conversationPublicID),
		SelectedToolIDs:      []uint{},
		ConfirmedToolIDs:     []uint{},
	}, nil
}

func (s *Service) UpsertToolPreference(ctx context.Context, userID uint, input ToolPreferenceInput) (*domainmcp.ToolPreference, error) {
	selectedIDs := uniqueToolIDs(input.SelectedToolIDs)
	confirmedIDs := uniqueToolIDs(input.ConfirmedToolIDs)
	if len(selectedIDs) > s.resolveMaxSelectedToolsPerMessage() {
		return nil, ErrInvalidToolSelection
	}
	if len(selectedIDs) > 0 {
		tools, err := s.repo.ListToolsByIDsForUser(ctx, selectedIDs, userID)
		if err != nil {
			return nil, err
		}
		if len(tools) != len(selectedIDs) {
			return nil, ErrInvalidToolSelection
		}
	}
	return s.repo.UpsertToolPreference(ctx, repository.UpsertMCPToolPreferenceInput{
		UserID:               userID,
		ConversationPublicID: normalizeConversationPublicID(input.ConversationPublicID),
		SelectedToolIDs:      selectedIDs,
		ConfirmedToolIDs:     intersectUintList(confirmedIDs, selectedIDs),
		WebSearchEnabled:     input.WebSearchEnabled,
		CodeSandboxEnabled:   input.CodeSandboxEnabled,
		ResearchMaxLLMCalls:  clampResearchLLMCalls(input.ResearchMaxLLMCalls),
		ResearchMaxToolCalls: clampResearchToolCalls(input.ResearchMaxToolCalls),
	})
}

func (s *Service) TestServerConnection(ctx context.Context, serverID uint) ConnectionTestResult {
	server, err := s.repo.GetServer(ctx, serverID)
	if err != nil {
		return friendlyConnectionError(err)
	}
	return s.testServerConnection(ctx, server)
}

func (s *Service) TestUserServerConnection(ctx context.Context, userID uint, serverID uint) ConnectionTestResult {
	server, err := s.repo.GetServerForUser(ctx, serverID, userID, false)
	if err != nil {
		return friendlyConnectionError(ErrServerNotFound)
	}
	return s.testServerConnection(ctx, server)
}

func (s *Service) testServerConnection(ctx context.Context, server *domainmcp.Server) ConnectionTestResult {
	if server == nil {
		return friendlyConnectionError(ErrServerNotFound)
	}
	if err := s.validateServerBaseURL(server.BaseURL); err != nil {
		return friendlyConnectionError(ErrInvalidServerBaseURL)
	}
	if s.client == nil {
		return friendlyConnectionError(ErrMCPClientUnavailable)
	}
	token, err := s.decryptToken(server.AuthTokenEnc)
	if err != nil {
		return friendlyConnectionError(err)
	}
	headers, err := parseHeadersJSON(server.HeadersJSON)
	if err != nil {
		return friendlyConnectionError(err)
	}
	tools, err := s.client.ListTools(ctx, inframcp.CallConfig{
		BaseURL:   server.BaseURL,
		AuthToken: token,
		TimeoutMS: resolveServerTimeoutMS(server.TimeoutSeconds, s.cfg.Snapshot().MCPToolTimeoutSeconds, defaultMCPServerToolListTimeoutMS),
		Headers:   headers,
	})
	if err != nil {
		return friendlyConnectionError(err)
	}
	return ConnectionTestResult{
		OK:        true,
		ErrorCode: "",
		Message:   "connection_ok",
		ToolCount: len(tools),
	}
}

func (s *Service) StartServerOAuth(ctx context.Context, userID uint, serverID uint, redirectURI string) (OAuthStartResult, error) {
	server, err := s.repo.GetServerForUser(ctx, serverID, userID, false)
	if err != nil {
		return OAuthStartResult{}, ErrServerNotFound
	}
	authURL := strings.TrimSpace(server.OAuthAuthURL)
	clientID := strings.TrimSpace(server.OAuthClientID)
	if authURL == "" || clientID == "" {
		return OAuthStartResult{}, ErrInvalidServerBaseURL
	}
	parsed, err := url.Parse(authURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return OAuthStartResult{}, ErrInvalidServerBaseURL
	}
	state := randomOAuthState(userID, serverID)
	query := parsed.Query()
	query.Set("response_type", "code")
	query.Set("client_id", clientID)
	if redirect := strings.TrimSpace(redirectURI); redirect != "" {
		query.Set("redirect_uri", redirect)
	}
	if scopes := strings.TrimSpace(server.OAuthScopes); scopes != "" {
		query.Set("scope", scopes)
	}
	query.Set("state", state)
	parsed.RawQuery = query.Encode()
	return OAuthStartResult{AuthorizationURL: parsed.String(), State: state}, nil
}

func (s *Service) CompleteOAuthCallback(ctx context.Context, userID uint, serverID uint, code string, state string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" {
		return ErrInvalidServerBaseURL
	}
	if _, err := s.repo.GetServerForUser(ctx, serverID, userID, false); err != nil {
		return ErrServerNotFound
	}
	status := "pending_token_exchange"
	_, err := s.repo.UpdateServer(ctx, serverID, repository.UpdateMCPServerInput{OAuthStatus: &status})
	return err
}

func (s *Service) VoiceConfig() VoiceConfigResult {
	cfg := s.cfg.Snapshot()
	return VoiceConfigResult{
		ASREnabled:  cfg.VoiceASREnabled,
		ASRProvider: strings.TrimSpace(cfg.VoiceASRProvider),
		ASRModel:    strings.TrimSpace(cfg.VoiceASRModel),
		TTSEnabled:  cfg.VoiceTTSEnabled,
		TTSProvider: strings.TrimSpace(cfg.VoiceTTSProvider),
		TTSModel:    strings.TrimSpace(cfg.VoiceTTSModel),
		TTSVoice:    strings.TrimSpace(cfg.VoiceTTSVoice),
	}
}

func (s *Service) ReorderServersWithTools(ctx context.Context, order []ReorderServerInput) ([]domainmcp.ServerWithTools, error) {
	if len(order) == 0 {
		return nil, ErrInvalidToolSelection
	}
	currentServers, err := s.repo.ListServers(ctx)
	if err != nil {
		return nil, err
	}
	if len(order) != len(currentServers) {
		return nil, ErrInvalidToolSelection
	}

	allowedServers := make(map[uint]struct{}, len(currentServers))
	for _, server := range currentServers {
		allowedServers[server.ID] = struct{}{}
	}
	seenServers := make(map[uint]struct{}, len(order))
	for _, item := range order {
		if item.ServerID == 0 {
			return nil, ErrInvalidToolSelection
		}
		if _, ok := allowedServers[item.ServerID]; !ok {
			return nil, ErrInvalidToolSelection
		}
		if _, ok := seenServers[item.ServerID]; ok {
			return nil, ErrInvalidToolSelection
		}
		seenServers[item.ServerID] = struct{}{}

		currentTools, err := s.repo.ListTools(ctx, item.ServerID, false)
		if err != nil {
			return nil, err
		}
		if len(item.ToolIDs) != len(currentTools) {
			return nil, ErrInvalidToolSelection
		}
		allowedTools := make(map[uint]struct{}, len(currentTools))
		for _, tool := range currentTools {
			allowedTools[tool.ID] = struct{}{}
		}
		seenTools := make(map[uint]struct{}, len(item.ToolIDs))
		for _, toolID := range item.ToolIDs {
			if toolID == 0 {
				return nil, ErrInvalidToolSelection
			}
			if _, ok := allowedTools[toolID]; !ok {
				return nil, ErrInvalidToolSelection
			}
			if _, ok := seenTools[toolID]; ok {
				return nil, ErrInvalidToolSelection
			}
			seenTools[toolID] = struct{}{}
		}
	}
	repoOrder := make([]repository.ReorderMCPServerInput, 0, len(order))
	for _, item := range order {
		repoOrder = append(repoOrder, repository.ReorderMCPServerInput{
			ServerID: item.ServerID,
			ToolIDs:  item.ToolIDs,
		})
	}
	return s.repo.ReorderServersWithTools(ctx, repoOrder)
}

func (s *Service) normalizeServerInput(input ServerInput, requireToken bool) (ServerInput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len([]rune(name)) > 128 {
		return ServerInput{}, ErrInvalidServerName
	}
	baseURL := strings.TrimSpace(input.BaseURL)
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return ServerInput{}, ErrInvalidServerBaseURL
	}
	if err = s.validateServerBaseURL(baseURL); err != nil {
		return ServerInput{}, ErrInvalidServerBaseURL
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "active"
	}
	switch status {
	case "active", "inactive":
	default:
		return ServerInput{}, ErrInvalidServerStatus
	}
	headersJSON := strings.TrimSpace(input.HeadersJSON)
	if headersJSON == "" {
		headersJSON = "{}"
	}
	if _, err = parseHeadersJSON(headersJSON); err != nil {
		return ServerInput{}, ErrInvalidServerHeaders
	}
	timeoutSeconds := input.TimeoutSeconds
	if timeoutSeconds < 0 || timeoutSeconds > maxServerTimeoutSeconds {
		return ServerInput{}, ErrInvalidServerTimeout
	}
	oauthAuthURL := strings.TrimSpace(input.OAuthAuthURL)
	if oauthAuthURL != "" {
		if parsed, parseErr := url.Parse(oauthAuthURL); parseErr != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return ServerInput{}, ErrInvalidServerBaseURL
		}
	}
	oauthTokenURL := strings.TrimSpace(input.OAuthTokenURL)
	if oauthTokenURL != "" {
		if parsed, parseErr := url.Parse(oauthTokenURL); parseErr != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return ServerInput{}, ErrInvalidServerBaseURL
		}
	}
	if requireToken {
		input.AuthToken = strings.TrimSpace(input.AuthToken)
	}
	return ServerInput{
		OwnerUserID:       input.OwnerUserID,
		Name:              name,
		BaseURL:           baseURL,
		AuthToken:         strings.TrimSpace(input.AuthToken),
		HeadersJSON:       headersJSON,
		Status:            status,
		TimeoutSeconds:    timeoutSeconds,
		OAuthClientID:     strings.TrimSpace(input.OAuthClientID),
		OAuthClientSecret: strings.TrimSpace(input.OAuthClientSecret),
		OAuthAuthURL:      oauthAuthURL,
		OAuthTokenURL:     oauthTokenURL,
		OAuthScopes:       strings.TrimSpace(input.OAuthScopes),
		OAuthAccessToken:  strings.TrimSpace(input.OAuthAccessToken),
		OAuthRefreshToken: strings.TrimSpace(input.OAuthRefreshToken),
	}, nil
}

func (s *Service) validateServerBaseURL(raw string) error {
	if s == nil {
		return ErrInvalidServerBaseURL
	}
	return security.ValidateTrustedOutboundHTTPURL(raw)
}

func normalizeToolInput(input ToolInput) (repository.UpdateMCPToolInput, error) {
	update := repository.UpdateMCPToolInput{}
	if input.DisplayName != nil {
		displayName := strings.TrimSpace(*input.DisplayName)
		if len([]rune(displayName)) > 160 {
			return update, ErrInvalidToolName
		}
		update.DisplayName = &displayName
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if len([]rune(description)) > 4096 {
			return update, ErrInvalidToolDesc
		}
		update.Description = &description
	}
	if input.Status != nil {
		status, err := normalizeToolStatus(*input.Status)
		if err != nil {
			return update, err
		}
		update.Status = &status
	}
	if input.AttachmentInputMode != nil {
		mode := strings.ToLower(strings.TrimSpace(*input.AttachmentInputMode))
		switch mode {
		case domainmcp.AttachmentInputModeNone, domainmcp.AttachmentInputModeImage:
			update.AttachmentInputMode = &mode
		default:
			return update, ErrInvalidToolAttachmentConfig
		}
	}
	if input.AttachmentArgument != nil {
		value := strings.TrimSpace(*input.AttachmentArgument)
		if len([]rune(value)) > 128 {
			return update, ErrInvalidToolAttachmentConfig
		}
		update.AttachmentArgument = &value
	}
	if input.AttachmentEncoding != nil {
		encoding := strings.ToLower(strings.TrimSpace(*input.AttachmentEncoding))
		switch encoding {
		case "", domainmcp.AttachmentEncodingBase64, domainmcp.AttachmentEncodingDataURL:
			update.AttachmentEncoding = &encoding
		default:
			return update, ErrInvalidToolAttachmentConfig
		}
	}
	if input.AttachmentPromptArgument != nil {
		value := strings.TrimSpace(*input.AttachmentPromptArgument)
		if len([]rune(value)) > 128 {
			return update, ErrInvalidToolAttachmentConfig
		}
		update.AttachmentPromptArgument = &value
	}
	return update, nil
}

type toolAttachmentConfig struct {
	Mode           string
	Argument       string
	Encoding       string
	PromptArgument string
}

func toolAttachmentConfigChanged(input ToolInput) bool {
	return input.AttachmentInputMode != nil ||
		input.AttachmentArgument != nil ||
		input.AttachmentEncoding != nil ||
		input.AttachmentPromptArgument != nil
}

func mergedToolAttachmentConfig(tool domainmcp.Tool, update repository.UpdateMCPToolInput) toolAttachmentConfig {
	config := toolAttachmentConfig{
		Mode:           strings.TrimSpace(tool.AttachmentInputMode),
		Argument:       strings.TrimSpace(tool.AttachmentArgument),
		Encoding:       strings.TrimSpace(tool.AttachmentEncoding),
		PromptArgument: strings.TrimSpace(tool.AttachmentPromptArgument),
	}
	if update.AttachmentInputMode != nil {
		config.Mode = *update.AttachmentInputMode
	}
	if update.AttachmentArgument != nil {
		config.Argument = *update.AttachmentArgument
	}
	if update.AttachmentEncoding != nil {
		config.Encoding = *update.AttachmentEncoding
	}
	if update.AttachmentPromptArgument != nil {
		config.PromptArgument = *update.AttachmentPromptArgument
	}
	if update.AttachmentInputMode != nil && *update.AttachmentInputMode == domainmcp.AttachmentInputModeNone {
		config.Argument = ""
		config.Encoding = ""
		config.PromptArgument = ""
	}
	return config
}

func validateToolAttachmentConfig(config toolAttachmentConfig, schemaJSON string) error {
	if config.Mode == domainmcp.AttachmentInputModeNone {
		if config.Argument != "" || config.Encoding != "" || config.PromptArgument != "" {
			return ErrInvalidToolAttachmentConfig
		}
		return nil
	}
	if config.Mode != domainmcp.AttachmentInputModeImage || config.Argument == "" {
		return ErrInvalidToolAttachmentConfig
	}
	switch config.Encoding {
	case domainmcp.AttachmentEncodingBase64, domainmcp.AttachmentEncodingDataURL:
	default:
		return ErrInvalidToolAttachmentConfig
	}
	if config.PromptArgument == config.Argument {
		return ErrInvalidToolAttachmentConfig
	}

	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(schemaJSON)), &schema); err != nil || len(schema.Properties) == 0 {
		return ErrInvalidToolAttachmentConfig
	}
	if !toolSchemaPropertyAcceptsString(schemaJSON, config.Argument) {
		return ErrInvalidToolAttachmentConfig
	}
	if config.PromptArgument != "" && !toolSchemaPropertyAcceptsString(schemaJSON, config.PromptArgument) {
		return ErrInvalidToolAttachmentConfig
	}
	for _, name := range schema.Required {
		name = strings.TrimSpace(name)
		if name != "" && name != config.Argument && name != config.PromptArgument {
			return ErrInvalidToolAttachmentConfig
		}
	}
	return nil
}

func normalizeToolStatus(status string) (string, error) {
	normalized := strings.TrimSpace(status)
	switch normalized {
	case "active", "inactive":
		return normalized, nil
	default:
		return "", ErrInvalidToolStatus
	}
}

func (s *Service) encryptToken(token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", nil
	}
	return secretbox.EncryptString(s.cfg.Snapshot().DataEncryptionKey, token)
}

func (s *Service) encryptOptionalToken(token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", nil
	}
	return s.encryptToken(token)
}

func (s *Service) decryptToken(encrypted string) (string, error) {
	if strings.TrimSpace(encrypted) == "" {
		return "", nil
	}
	return secretbox.DecryptString(s.cfg.Snapshot().DataEncryptionKey, encrypted)
}

func parseHeadersJSON(raw string) (map[string]string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return map[string]string{}, nil
	}
	payload := map[string]string{}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidServerHeaders, err)
	}
	result := make(map[string]string, len(payload))
	for key, item := range payload {
		headerKey := strings.TrimSpace(key)
		if headerKey == "" {
			continue
		}
		result[headerKey] = strings.TrimSpace(item)
	}
	return result, nil
}

func resolveServerTimeoutMS(serverSeconds int, fallbackSeconds int, hardDefaultMS int) int {
	if serverSeconds > 0 {
		return serverSeconds * 1000
	}
	if fallbackSeconds > 0 {
		return fallbackSeconds * 1000
	}
	return hardDefaultMS
}

func (s *Service) resolveMaxSelectedToolsPerMessage() int {
	if s == nil || s.cfg == nil {
		return config.DefaultMCPMaxSelectedToolsPerMessage
	}
	maxTools := s.cfg.Snapshot().MCPMaxSelectedToolsPerMessage
	if maxTools <= 0 {
		maxTools = config.DefaultMCPMaxSelectedToolsPerMessage
	}
	if maxTools > config.MaxMCPSelectedToolsPerMessage {
		maxTools = config.MaxMCPSelectedToolsPerMessage
	}
	return maxTools
}

func normalizeConversationPublicID(value string) string {
	return strings.TrimSpace(value)
}

func uniqueToolIDs(items []uint) []uint {
	seen := make(map[uint]struct{}, len(items))
	result := make([]uint, 0, len(items))
	for _, item := range items {
		if item == 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func intersectUintList(items []uint, allowed []uint) []uint {
	allowedSet := make(map[uint]struct{}, len(allowed))
	for _, item := range allowed {
		if item != 0 {
			allowedSet[item] = struct{}{}
		}
	}
	result := make([]uint, 0, len(items))
	for _, item := range uniqueToolIDs(items) {
		if _, ok := allowedSet[item]; ok {
			result = append(result, item)
		}
	}
	return result
}

func clampResearchLLMCalls(value int) int {
	if value <= 0 {
		return 0
	}
	if value < 2 {
		return 2
	}
	if value > 32 {
		return 32
	}
	return value
}

func clampResearchToolCalls(value int) int {
	if value <= 0 {
		return 0
	}
	if value > 64 {
		return 64
	}
	return value
}

func friendlyConnectionError(err error) ConnectionTestResult {
	message := strings.TrimSpace(err.Error())
	code := "connection_failed"
	switch {
	case errors.Is(err, ErrServerNotFound):
		code = "not_found"
		message = "not_found"
	case errors.Is(err, ErrInvalidServerBaseURL):
		code = "invalid_url"
		message = "invalid_url"
	case errors.Is(err, ErrMCPClientUnavailable):
		code = "client_unavailable"
		message = "client_unavailable"
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(strings.ToLower(message), "timeout"), strings.Contains(strings.ToLower(message), "deadline"):
		code = "timeout"
		message = "timeout"
	case strings.Contains(strings.ToLower(message), "status=401"), strings.Contains(strings.ToLower(message), "status=403"), strings.Contains(strings.ToLower(message), "unauthorized"):
		code = "unauthorized"
		message = "unauthorized"
	case strings.Contains(strings.ToLower(message), "json-rpc"), strings.Contains(strings.ToLower(message), "protocol"), strings.Contains(strings.ToLower(message), "event stream"):
		code = "protocol_mismatch"
		message = "protocol_mismatch"
	}
	return ConnectionTestResult{
		OK:        false,
		ErrorCode: code,
		Message:   message,
	}
}

func randomOAuthState(userID uint, serverID uint) string {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("u%d-s%d-%d", userID, serverID, time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(raw[:])
}
