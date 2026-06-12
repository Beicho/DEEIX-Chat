package mcp

import "time"

type ServerResponse struct {
	ID              uint       `json:"id"`
	OwnerUserID     uint       `json:"ownerUserID"`
	Scope           string     `json:"scope"`
	Name            string     `json:"name"`
	BaseURL         string     `json:"baseURL"`
	HeadersJSON     string     `json:"headersJSON"`
	Status          string     `json:"status"`
	TimeoutSeconds  int        `json:"timeoutSeconds"`
	OAuthClientID   string     `json:"oauthClientID"`
	OAuthAuthURL    string     `json:"oauthAuthURL"`
	OAuthTokenURL   string     `json:"oauthTokenURL"`
	OAuthScopes     string     `json:"oauthScopes"`
	OAuthStatus     string     `json:"oauthStatus"`
	ToolCount       int        `json:"toolCount"`
	ActiveToolCount int        `json:"activeToolCount"`
	LastSyncedAt    *time.Time `json:"lastSyncedAt"`
	LastError       string     `json:"lastError"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type ToolResponse struct {
	ID              uint      `json:"id"`
	ServerID        uint      `json:"serverID"`
	ServerName      string    `json:"serverName"`
	Name            string    `json:"name"`
	DisplayName     string    `json:"displayName"`
	Description     string    `json:"description"`
	InputSchemaJSON string    `json:"inputSchemaJSON"`
	Status          string    `json:"status"`
	DefaultEnabled  bool      `json:"defaultEnabled"`
	RequiresConfirm bool      `json:"requiresConfirmation"`
	ToolKind        string    `json:"toolKind"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type CreateServerRequest struct {
	Name              string `json:"name"`
	BaseURL           string `json:"baseURL"`
	AuthToken         string `json:"authToken"`
	HeadersJSON       string `json:"headersJSON"`
	Status            string `json:"status"`
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
	DisplayName     *string `json:"displayName"`
	Description     *string `json:"description"`
	Status          *string `json:"status"`
	DefaultEnabled  *bool   `json:"defaultEnabled"`
	RequiresConfirm *bool   `json:"requiresConfirmation"`
}

type UpdateServerToolsStatusRequest struct {
	ToolIDs []uint `json:"toolIDs"`
	Status  string `json:"status"`
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
