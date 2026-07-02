package repository

import (
	"context"

	domainmcp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/mcp"
)

// CreateMCPServerInput 定义创建 MCP 服务字段。
type CreateMCPServerInput struct {
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
}

// UpdateMCPServerInput 定义更新 MCP 服务字段。
type UpdateMCPServerInput struct {
	Name                 *string
	BaseURL              *string
	AuthTokenEnc         *string
	HeadersJSON          *string
	Status               *string
	TimeoutSeconds       *int
	OAuthClientID        *string
	OAuthClientSecretEnc *string
	OAuthAuthURL         *string
	OAuthTokenURL        *string
	OAuthScopes          *string
	OAuthAccessTokenEnc  *string
	OAuthRefreshTokenEnc *string
	OAuthStatus          *string
	LastError            *string
}

// UpdateMCPToolInput 定义更新 MCP 工具字段。
type UpdateMCPToolInput struct {
	DisplayName     *string
	Description     *string
	Status          *string
	DefaultEnabled  *bool
	RequiresConfirm *bool
}

// UpsertMCPToolPreferenceInput 定义用户工具选择偏好。
type UpsertMCPToolPreferenceInput struct {
	UserID               uint
	ConversationPublicID string
	SelectedToolIDs      []uint
	ConfirmedToolIDs     []uint
	WebSearchEnabled     bool
	CodeSandboxEnabled   bool
	ResearchMaxLLMCalls  int
	ResearchMaxToolCalls int
}

type ReorderMCPServerInput struct {
	ServerID uint
	ToolIDs  []uint
}

// MCPRepository 封装 MCP 控制面持久化。
type MCPRepository interface {
	CreateServer(ctx context.Context, input CreateMCPServerInput) (*domainmcp.Server, error)
	UpdateServer(ctx context.Context, serverID uint, input UpdateMCPServerInput) (*domainmcp.Server, error)
	ListServers(ctx context.Context) ([]domainmcp.Server, error)
	ListServersForUser(ctx context.Context, userID uint, includePlatform bool) ([]domainmcp.Server, error)
	GetServer(ctx context.Context, serverID uint) (*domainmcp.Server, error)
	GetServerForUser(ctx context.Context, serverID uint, userID uint, includePlatform bool) (*domainmcp.Server, error)
	DeleteServer(ctx context.Context, serverID uint) error
	ReplaceServerTools(ctx context.Context, serverID uint, tools []domainmcp.Tool) error
	ListTools(ctx context.Context, serverID uint, onlyActive bool) ([]domainmcp.Tool, error)
	ListToolsByIDs(ctx context.Context, toolIDs []uint) ([]domainmcp.Tool, error)
	ListToolsByIDsForUser(ctx context.Context, toolIDs []uint, userID uint) ([]domainmcp.Tool, error)
	UpdateTool(ctx context.Context, toolID uint, input UpdateMCPToolInput) (*domainmcp.Tool, error)
	UpdateServerToolsStatus(ctx context.Context, serverID uint, toolIDs []uint, status string) ([]domainmcp.Tool, error)
	GetToolPreference(ctx context.Context, userID uint, conversationPublicID string) (*domainmcp.ToolPreference, error)
	UpsertToolPreference(ctx context.Context, input UpsertMCPToolPreferenceInput) (*domainmcp.ToolPreference, error)
	ReorderServersWithTools(ctx context.Context, order []ReorderMCPServerInput) ([]domainmcp.ServerWithTools, error)
}
