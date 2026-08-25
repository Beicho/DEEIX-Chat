package model

import "time"

// MCPServer 存储平台或用户配置的 MCP 服务。
type MCPServer struct {
	ControlPlaneModel
	OwnerUserID          uint       `gorm:"not null;default:0;index:idx_mcp_servers_owner;comment:归属用户ID，0表示平台级"`
	Name                 string     `gorm:"size:128;not null;default:'';comment:MCP服务名称"`
	BaseURL              string     `gorm:"size:512;not null;default:'';comment:MCP服务地址"`
	AuthTokenEnc         string     `gorm:"type:text;not null;default:'';comment:加密后的鉴权Token"`
	HeadersJSON          string     `gorm:"type:text;not null;default:'{}';comment:附加请求头JSON"`
	Status               string     `gorm:"size:32;not null;default:'active';index:idx_mcp_servers_status;comment:服务状态(active/inactive)"`
	SortOrder            int        `gorm:"not null;default:0;index:idx_mcp_servers_sort_order;comment:展示顺序"`
	TimeoutSeconds       int        `gorm:"not null;default:0;comment:单服务调用超时秒数"`
	OAuthClientID        string     `gorm:"size:255;not null;default:'';comment:OAuth客户端ID"`
	OAuthClientSecretEnc string     `gorm:"type:text;not null;default:'';comment:加密后的OAuth客户端密钥"`
	OAuthAuthURL         string     `gorm:"size:512;not null;default:'';comment:OAuth授权地址"`
	OAuthTokenURL        string     `gorm:"size:512;not null;default:'';comment:OAuth令牌地址"`
	OAuthScopes          string     `gorm:"size:512;not null;default:'';comment:OAuth授权范围"`
	OAuthAccessTokenEnc  string     `gorm:"type:text;not null;default:'';comment:加密后的OAuth访问令牌"`
	OAuthRefreshTokenEnc string     `gorm:"type:text;not null;default:'';comment:加密后的OAuth刷新令牌"`
	OAuthTokenExpiresAt  *time.Time `gorm:"comment:OAuth访问令牌过期时间"`
	OAuthStatus          string     `gorm:"size:32;not null;default:'';comment:OAuth连接状态"`
	ToolCount            int        `gorm:"not null;default:0;comment:最近发现工具数量"`
	LastSyncedAt         *time.Time `gorm:"comment:最近同步工具时间"`
	LastError            string     `gorm:"type:text;not null;default:'';comment:最近同步或调用错误"`
}

func (MCPServer) TableName() string {
	return "mcp_servers"
}

// MCPTool 存储 MCP 服务发现的工具。
type MCPTool struct {
	ControlPlaneModel
	ServerID                 uint   `gorm:"not null;default:0;uniqueIndex:idx_mcp_tools_server_name,priority:1;index:idx_mcp_tools_server_id;comment:MCP服务ID"`
	Name                     string `gorm:"size:160;not null;default:'';uniqueIndex:idx_mcp_tools_server_name,priority:2;comment:工具名称"`
	DisplayName              string `gorm:"size:160;not null;default:'';comment:展示名称"`
	Description              string `gorm:"type:text;not null;default:'';comment:工具说明"`
	MetadataCustomized       *bool  `gorm:"comment:名称或说明是否由管理员修改(NULL表示升级前状态待确认)"`
	InputSchemaJSON          string `gorm:"type:text;not null;default:'{}';comment:输入JSON Schema"`
	AttachmentInputMode      string `gorm:"size:32;not null;default:'none';comment:附件输入模式(none/image)"`
	AttachmentArgument       string `gorm:"size:128;not null;default:'';comment:附件内容对应的顶层参数名"`
	AttachmentEncoding       string `gorm:"size:32;not null;default:'';comment:附件编码(base64/data_url)"`
	AttachmentPromptArgument string `gorm:"size:128;not null;default:'';comment:用户提示词对应的顶层参数名"`
	Status                   string `gorm:"size:32;not null;default:'inactive';index:idx_mcp_tools_status;comment:工具状态(active/inactive)"`
	SortOrder                int    `gorm:"not null;default:0;index:idx_mcp_tools_sort_order;comment:展示顺序"`
}

func (MCPTool) TableName() string {
	return "mcp_tools"
}

// MCPToolPreference 存储用户默认或单会话工具选择偏好。
type MCPToolPreference struct {
	ControlPlaneModel
	UserID               uint   `gorm:"not null;default:0;uniqueIndex:idx_mcp_tool_pref_user_conversation,priority:1;index:idx_mcp_tool_pref_user;comment:用户ID"`
	ConversationPublicID string `gorm:"size:64;not null;default:'';uniqueIndex:idx_mcp_tool_pref_user_conversation,priority:2;comment:会话公开ID，空串表示用户默认"`
	SelectedToolIDsJSON  string `gorm:"type:text;not null;default:'[]';comment:已选择工具ID JSON"`
	ConfirmedToolIDsJSON string `gorm:"type:text;not null;default:'[]';comment:已确认工具ID JSON"`
	WebSearchEnabled     bool   `gorm:"not null;default:false;comment:是否启用联网搜索"`
	CodeSandboxEnabled   bool   `gorm:"not null;default:false;comment:是否启用代码运行"`
	ResearchMaxLLMCalls  int    `gorm:"not null;default:0;comment:本轮研究最大模型请求次数"`
	ResearchMaxToolCalls int    `gorm:"not null;default:0;comment:本轮研究最大工具调用次数"`
}

func (MCPToolPreference) TableName() string {
	return "mcp_tool_preferences"
}
