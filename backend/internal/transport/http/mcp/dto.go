package mcp

import "time"

type ServerResponse struct {
	ID                                   uint       `json:"id"`
	OwnerUserID                          uint       `json:"ownerUserID"`
	Scope                                string     `json:"scope"`
	Name                                 string     `json:"name"`
	BaseURL                              string     `json:"baseURL"`
	HeadersJSON                          string     `json:"headersJSON"`
	Status                               string     `json:"status"`
	SortOrder                            int        `json:"sortOrder"`
	TimeoutSeconds                       int        `json:"timeoutSeconds"`
	OAuthClientID                        string     `json:"oauthClientID"`
	OAuthAuthURL                         string     `json:"oauthAuthURL"`
	OAuthTokenURL                        string     `json:"oauthTokenURL"`
	OAuthScopes                          string     `json:"oauthScopes"`
	OAuthStatus                          string     `json:"oauthStatus"`
	ToolCount                            int        `json:"toolCount"`
	ActiveToolCount                      int        `json:"activeToolCount"`
	RequiresToolMetadataSyncConfirmation bool       `json:"requiresToolMetadataSyncConfirmation"`
	LastSyncedAt                         *time.Time `json:"lastSyncedAt" extensions:"x-nullable,!x-omitempty"`
	LastError                            string     `json:"lastError"`
	CreatedAt                            time.Time  `json:"createdAt"`
	UpdatedAt                            time.Time  `json:"updatedAt"`
}

type ToolResponse struct {
	ID                       uint      `json:"id"`
	ServerID                 uint      `json:"serverID"`
	ServerName               string    `json:"serverName"`
	Name                     string    `json:"name"`
	DisplayName              string    `json:"displayName"`
	Description              string    `json:"description"`
	InputSchemaJSON          string    `json:"inputSchemaJSON"`
	AttachmentInputMode      string    `json:"attachmentInputMode" enums:"none,image"`
	AttachmentArgument       string    `json:"attachmentArgument"`
	AttachmentEncoding       string    `json:"attachmentEncoding" enums:",base64,data_url"`
	AttachmentPromptArgument string    `json:"attachmentPromptArgument"`
	PriceNanousd             int64     `json:"priceNanousd"`
	Status                   string    `json:"status"`
	SortOrder                int       `json:"sortOrder"`
	DefaultEnabled           bool      `json:"defaultEnabled"`
	RequiresConfirm          bool      `json:"requiresConfirmation"`
	ToolKind                 string    `json:"toolKind"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

type CreateServerRequest struct {
	Name              string `json:"name"`
	BaseURL           string `json:"baseURL"`
	AuthToken         string `json:"authToken,omitempty"`
	HeadersJSON       string `json:"headersJSON,omitempty"`
	Status            string `json:"status,omitempty"`
	TimeoutSeconds    int    `json:"timeoutSeconds"`
	OAuthClientID     string `json:"oauthClientID"`
	OAuthClientSecret string `json:"oauthClientSecret"`
	OAuthAuthURL      string `json:"oauthAuthURL"`
	OAuthTokenURL     string `json:"oauthTokenURL"`
	OAuthScopes       string `json:"oauthScopes"`
	OAuthAccessToken  string `json:"oauthAccessToken"`
	OAuthRefreshToken string `json:"oauthRefreshToken"`
}

type UpdateToolRequest struct {
	DisplayName              *string `json:"displayName,omitempty"`
	Description              *string `json:"description,omitempty"`
	AttachmentInputMode      *string `json:"attachmentInputMode,omitempty" enums:"none,image"`
	AttachmentArgument       *string `json:"attachmentArgument,omitempty"`
	AttachmentEncoding       *string `json:"attachmentEncoding,omitempty" enums:"base64,data_url"`
	AttachmentPromptArgument *string `json:"attachmentPromptArgument,omitempty"`
	// PriceNanousd 单次调用价格（nano USD），0 表示不单独计费。
	PriceNanousd    *int64  `json:"priceNanousd,omitempty" minimum:"0"`
	Status          *string `json:"status,omitempty"`
	DefaultEnabled  *bool   `json:"defaultEnabled"`
	RequiresConfirm *bool   `json:"requiresConfirmation"`
}

type UpdateServerToolsStatusRequest struct {
	ToolIDs []uint `json:"toolIDs"`
	Status  string `json:"status"`
}

type ReorderServerOrderItem struct {
	ServerID uint   `json:"serverID"`
	ToolIDs  []uint `json:"toolIDs"`
}

type ReorderServersRequest struct {
	Servers []ReorderServerOrderItem `json:"servers"`
}

type ServerDataResponse struct {
	Server ServerResponse `json:"server"`
}

type DeleteServerResponse struct {
	Deleted bool `json:"deleted"`
}

type ServerListResponse struct {
	Results []ServerResponse `json:"results"`
}

type ToolListResponse struct {
	Results []ToolResponse `json:"results"`
}

type ToolPreferenceRequest struct {
	ConversationPublicID string `json:"conversationPublicID"`
	SelectedToolIDs      []uint `json:"selectedToolIDs"`
	ConfirmedToolIDs     []uint `json:"confirmedToolIDs"`
	WebSearchEnabled     bool   `json:"webSearchEnabled"`
	CodeSandboxEnabled   bool   `json:"codeSandboxEnabled"`
	ResearchMaxLLMCalls  int    `json:"researchMaxLLMCalls"`
	ResearchMaxToolCalls int    `json:"researchMaxToolCalls"`
}

type ToolPreferenceResponse struct {
	ConversationPublicID string    `json:"conversationPublicID"`
	SelectedToolIDs      []uint    `json:"selectedToolIDs"`
	ConfirmedToolIDs     []uint    `json:"confirmedToolIDs"`
	WebSearchEnabled     bool      `json:"webSearchEnabled"`
	CodeSandboxEnabled   bool      `json:"codeSandboxEnabled"`
	ResearchMaxLLMCalls  int       `json:"researchMaxLLMCalls"`
	ResearchMaxToolCalls int       `json:"researchMaxToolCalls"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type ConnectionTestResponse struct {
	OK        bool   `json:"ok"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
	ToolCount int    `json:"toolCount"`
}

type OAuthStartRequest struct {
	RedirectURI string `json:"redirectURI"`
}

type OAuthStartResponse struct {
	AuthorizationURL string `json:"authorizationURL"`
	State            string `json:"state"`
}

type OAuthCallbackRequest struct {
	ServerID uint   `json:"serverID"`
	Code     string `json:"code"`
	State    string `json:"state"`
}

type OAuthCallbackResponse struct {
	Accepted bool   `json:"accepted"`
	Status   string `json:"status"`
}

type VoiceConfigResponse struct {
	ASREnabled  bool   `json:"asrEnabled"`
	ASRProvider string `json:"asrProvider"`
	ASRModel    string `json:"asrModel"`
	TTSEnabled  bool   `json:"ttsEnabled"`
	TTSProvider string `json:"ttsProvider"`
	TTSModel    string `json:"ttsModel"`
	TTSVoice    string `json:"ttsVoice"`
}

type ServerToolOrderResponse struct {
	Server ServerResponse `json:"server"`
	Tools  []ToolResponse `json:"tools"`
}

type ServerToolOrderListResponse struct {
	Results []ServerToolOrderResponse `json:"results"`
}

// ErrorDoc 表示 MCP 管理接口的错误响应。
type ErrorDoc struct {
	ErrorMsg string `json:"errorMsg"`
}

// ServerListResponseDoc 包裹 MCP 服务列表响应。
type ServerListResponseDoc struct {
	ErrorMsg string             `json:"errorMsg"`
	Data     ServerListResponse `json:"data"`
}

// ServerDataResponseDoc 包裹 MCP 服务详情响应。
type ServerDataResponseDoc struct {
	ErrorMsg string             `json:"errorMsg"`
	Data     ServerDataResponse `json:"data"`
}

// ToolListResponseDoc 包裹 MCP 工具列表响应。
type ToolListResponseDoc struct {
	ErrorMsg string           `json:"errorMsg"`
	Data     ToolListResponse `json:"data"`
}

// ToolResponseDoc 包裹 MCP 工具详情响应。
type ToolResponseDoc struct {
	ErrorMsg string       `json:"errorMsg"`
	Data     ToolResponse `json:"data"`
}

// ServerToolOrderListResponseDoc 包裹 MCP 服务及工具排序响应。
type ServerToolOrderListResponseDoc struct {
	ErrorMsg string                      `json:"errorMsg"`
	Data     ServerToolOrderListResponse `json:"data"`
}

// DeleteServerResponseDoc 包裹 MCP 服务删除响应。
type DeleteServerResponseDoc struct {
	ErrorMsg string               `json:"errorMsg"`
	Data     DeleteServerResponse `json:"data"`
}
