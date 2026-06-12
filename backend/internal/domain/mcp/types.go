package mcp

import "time"

// Server 表示平台或用户维护的 MCP 服务。
type Server struct {
	ID                   uint
	OwnerUserID          uint
	Name                 string
	BaseURL              string
	AuthTokenEnc         string
	HeadersJSON          string
	Status               string
	TimeoutSeconds       int
	OAuthClientID        string
	OAuthClientSecretEnc string
	OAuthAuthURL         string
	OAuthTokenURL        string
	OAuthScopes          string
	OAuthAccessTokenEnc  string
	OAuthRefreshTokenEnc string
	OAuthTokenExpiresAt  *time.Time
	OAuthStatus          string
	ToolCount            int
	ActiveToolCount      int
	LastSyncedAt         *time.Time
	LastError            string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Tool 表示从 MCP 服务发现并由管理员控制可用性的工具。
type Tool struct {
	ID              uint
	ServerID        uint
	ServerName      string
	Name            string
	DisplayName     string
	Description     string
	InputSchemaJSON string
	Status          string
	DefaultEnabled  bool
	RequiresConfirm bool
	ToolKind        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ToolPreference 表示用户默认或单会话工具选择。
type ToolPreference struct {
	ID                   uint
	UserID               uint
	ConversationPublicID string
	SelectedToolIDs      []uint
	ConfirmedToolIDs     []uint
	WebSearchEnabled     bool
	CodeSandboxEnabled   bool
	ResearchMaxLLMCalls  int
	ResearchMaxToolCalls int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
