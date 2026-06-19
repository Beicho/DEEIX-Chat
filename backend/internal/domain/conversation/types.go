package conversation

import "time"

// Conversation 表示会话元信息。
type Conversation struct {
	ID                    uint
	UserID                uint
	ProjectID             *uint
	ProjectPublicID       string
	ProjectName           string
	ProjectSystemPrompt   string
	PublicID              string
	Title                 string
	LabelsJSON            string
	Model                 string
	Provider              string
	SessionKey            string
	IsStarred             bool
	StarredAt             *time.Time
	MessageCount          int
	Status                string
	ContextPolicy         string
	LastCompactedAt       *time.Time
	LastResponseID        string
	LastPromptFingerprint string
	ShareStatus           string
	ShareID               string
	SharedAt              *time.Time
	LastShareAccessedAt   *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ConversationSearchResult 表示一次会话全文搜索命中。
type ConversationSearchResult struct {
	Conversation    Conversation
	MessagePublicID string
	MessageRole     string
	MessageSnippet  string
	MatchedTitle    bool
	MatchedAt       time.Time
}

// ConversationProject 表示用户会话项目分组。
type ConversationProject struct {
	ID           uint
	UserID       uint
	PublicID     string
	Name         string
	Description  string
	SystemPrompt string
	Color        string
	Icon         string
	SortOrder    int
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ProjectDocument 表示项目级资料库文档引用。
type ProjectDocument struct {
	ID            uint
	UserID        uint
	ProjectID     uint
	FileObjID     uint
	FileID        string
	FileName      string
	FileSize      int64
	FileCategory  string
	ExtractStatus string
	EmbedStatus   string
	IndexStatus   string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ConversationProjectPatch 表示项目分组的局部更新。
type ConversationProjectPatch struct {
	Name         *string
	Description  *string
	SystemPrompt *string
	Color        *string
	Icon         *string
	Status       *string
}

// ConversationShare 表示会话公开分享快照。
type ConversationShare struct {
	ID                    uint
	ShareID               string
	ConversationID        uint
	UserID                uint
	Status                string
	TitleSnapshot         string
	ModelSnapshot         string
	MessageIDsJSON        string
	DefaultMessageIDsJSON string
	ShareScope            string
	PasswordHash          string
	IncludeThinking       bool
	ExpiresAt             *time.Time
	RevokedAt             *time.Time
	RegeneratedAt         *time.Time
	LastAccessedAt        *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// MessageTraceBlock 表示单个消息轨迹块。
type MessageTraceBlock struct {
	Title           string
	Summary         string
	ContentMarkdown string
	Status          string
	Stage           string
	RoundID         string
	ParentEventID   string
	UpdatedAt       time.Time
	PayloadJSON     string
}

// MessageTraceEvent 表示按发生顺序记录的消息轨迹事件。
type MessageTraceEvent struct {
	EventID         string
	EventType       string
	Phase           string
	Stage           string
	RoundID         string
	ParentEventID   string
	Title           string
	Summary         string
	ContentMarkdown string
	Status          string
	Seq             int
	StartedAt       time.Time
	EndedAt         *time.Time
	UpdatedAt       time.Time
	PayloadJSON     string
}

// MessageProcessTrace 表示消息处理、工具调用与上游 think 聚合结果。
type MessageProcessTrace struct {
	Enabled       bool
	Status        string
	Process       *MessageTraceBlock
	Tools         *MessageTraceBlock
	UpstreamThink *MessageTraceBlock
	PromptTrace   *MessagePromptTrace
	Events        []MessageTraceEvent
}

// MessagePromptTraceBlock 表示一次上游请求中的上下文规划块。
type MessagePromptTraceBlock struct {
	Kind          string
	Title         string
	TokenEstimate int64
	Cacheable     bool
	SourceCount   int
	SourceRefs    []MessagePromptTraceSourceRef
}

// MessagePromptTraceSourceRef 表示 PromptTrace 中的上下文来源引用。
type MessagePromptTraceSourceRef struct {
	SourceType string
	SourceID   string
	Title      string
	ArtifactID uint
}

// MessagePromptTrace 表示本轮请求发送前的 PromptPlan 摘要。
type MessagePromptTrace struct {
	Mode                   string
	PromptFingerprint      string
	StatefulUsed           bool
	StatefulDisabledReason string
	TotalTokenEstimate     int64
	SentTokenEstimate      int64
	FullMessageCount       int
	SentMessageCount       int
	StatefulSavedMessages  int
	StatefulSavedTokens    int64
	Blocks                 []MessagePromptTraceBlock
}

// Message 表示会话消息。
type Message struct {
	ID               uint
	ConversationID   uint
	UserID           uint
	PublicID         string
	ParentMessageID  *uint
	RunID            string
	Role             string
	ContentType      string
	Content          string
	BranchReason     string
	MessageGroupID   string
	SourceMessageID  *uint
	TokenUsage       int64
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ReasoningTokens  int64
	LatencyMS        int64
	BilledCurrency   string
	BilledNanousd    int64
	PricingSnapshot  string
	Status           string
	ErrorCode        string
	ErrorMessage     string
	Attachments      string
	ParentPublicID   string
	SourcePublicID   string
	MyFeedback       string
	ThumbsUpCount    int64
	ThumbsDownCount  int64
	Bookmarked       bool
	ProcessTrace     *MessageProcessTrace
	EditedAt         *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// MessageBookmark 表示用户收藏的一条消息。
type MessageBookmark struct {
	ID             uint
	UserID         uint
	ConversationID uint
	MessageID      uint
	Note           string
	TagsJSON       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// MessageBookmarkListItem 表示收藏列表中的一项。
type MessageBookmarkListItem struct {
	Bookmark     MessageBookmark
	Conversation Conversation
	Message      Message
}

// ConversationDraft 表示用户输入框草稿快照。
type ConversationDraft struct {
	ID                   uint
	UserID               uint
	ConversationPublicID string
	Draft                string
	AttachmentsJSON      string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// MessageFeedback 表示消息反馈。
type MessageFeedback struct {
	ID             uint
	UserID         uint
	ConversationID uint
	MessageID      uint
	Feedback       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Attachment 表示附件元信息。
type Attachment struct {
	ID             uint
	ConversationID uint
	MessageID      uint
	UserID         uint
	FileID         string
	Kind           string
	FileName       string
	MimeType       string
	FileSize       int64
	SHA256         string
	StoragePath    string
	Status         string
	MetaJSON       string
	UploadedAt     time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FileObject 表示文件对象。
type FileObject struct {
	ID                     uint
	FileID                 string
	UserID                 uint
	Purpose                string
	FileName               string
	MimeType               string
	DetectedMIME           string
	FileCategory           string
	SizeBytes              int64
	SHA256                 string
	StoragePath            string
	Status                 string
	LastAccessedAt         *time.Time
	ExpiresAt              *time.Time
	ProcessingStatus       string
	ProcessingReady        bool
	ProcessingErrorCode    string
	ProcessingErrorMessage string
	ExtractStatus          string
	ExtractEngine          string
	ExtractStoragePath     string
	ExtractChars           int
	ExtractPages           int
	PreviewText            string
	OCRUsed                bool
	RAGReady               bool
	RAGReason              string
	EmbedStatus            string
	EmbedError             string
	PageCount              int
	ChunkCount             int
	ExtractorVersion       string
	ExtractedAt            *time.Time
	ProcessingPayloadJSON  string
	ProcessingStartedAt    *time.Time
	ProcessingCompletedAt  *time.Time
	RagOptOut              bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// FileObjectProcessing 表示 file_objects 中的服务端处理状态。
type FileObjectProcessing struct {
	ID                 uint
	FileObjectID       uint
	UserID             uint
	DetectedMIME       string
	FileCategory       string
	ProcessingStatus   string
	ExtractStatus      string
	ExtractEngine      string
	ExtractStoragePath string
	ExtractChars       int
	ExtractPages       int
	PreviewText        string
	OCRUsed            bool
	RAGReady           bool
	RAGReason          string
	ErrorCode          string
	ErrorMessage       string
	ExtractorVersion   string
	PayloadJSON        string
	StartedAt          *time.Time
	CompletedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// FileChunk 表示文件分片。
type FileChunk struct {
	ID         uint
	FileObjID  uint
	UserID     uint
	ChunkIndex int
	PageNum    int
	CharOffset int
	Content    string
	TokenCount int
	CreatedAt  time.Time
}

// FileChunkSearchResult 表示分片检索结果。
type FileChunkSearchResult struct {
	FileChunk
	Similarity float32
}

// StorageQuota 表示用户文件配额。
type StorageQuota struct {
	ID            uint
	UserID        uint
	QuotaBytes    int64
	UsedBytes     int64
	ReservedBytes int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Run 表示对话运行日志。
type Run struct {
	ID                  uint
	RunID               string
	RequestID           string
	UserID              uint
	ConversationID      uint
	TaskType            string
	Endpoint            string
	Provider            string
	ProviderProtocol    string
	UpstreamID          uint
	UpstreamModelID     uint
	UpstreamName        string
	RequestedModelName  string
	PlatformModelName   string
	RoutedBindingCode   string
	ModelVendor         string
	ModelIcon           string
	UpstreamModelName   string
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheWriteTokens    int64
	ReasoningTokens     int64
	ToolCallsCount      int
	FirstTokenLatencyMS int64
	TotalLatencyMS      int64
	Status              string
	ErrorCode           string
	ErrorMessage        string
	StartedAt           time.Time
	EndedAt             *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ModelAvailability 表示公开状态页可展示的模型可用性聚合。
type ModelAvailability struct {
	ModelName   string
	CallCount   int64
	SuccessRate float64
	Status      string
}

// ModerationEvent 记录一次内容检查命中或检查失败。
type ModerationEvent struct {
	ID                    uint
	UserID                uint
	ConversationID        uint
	MessageID             uint
	RunID                 string
	Direction             string
	Action                string
	Model                 string
	Score                 float64
	Threshold             float64
	Flagged               bool
	CategoriesJSON        string
	Reason                string
	EventType             string
	ContentSnapshot       string
	ContentHash           string
	SnapshotTruncated     bool
	ReviewStatus          string
	ReviewedBy            uint
	ReviewedAt            *time.Time
	ReviewNote            string
	Disposition           string
	DispositionAppliedAt  *time.Time
	DispositionReleasedAt *time.Time
	DispositionReleasedBy uint
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ModerationEventFilter scopes the admin moderation event list.
type ModerationEventFilter struct {
	UserID       uint
	Direction    string
	ReviewStatus string
	Disposition  string
	EventType    string
	Flagged      *bool
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
}

// MessageTrace 表示消息处理轨迹。
type MessageTrace struct {
	ID              uint
	MessageID       uint
	ConversationID  uint
	UserID          uint
	RunID           string
	TraceType       string
	Status          string
	Stage           string
	RoundID         string
	ParentEventID   string
	Title           string
	Summary         string
	ContentMarkdown string
	PayloadJSON     string
	Seq             int
	StartedAt       time.Time
	EndedAt         *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// MessageTraceEventRow 表示消息轨迹事件持久化行。
type MessageTraceEventRow struct {
	ID              uint
	MessageID       uint
	ConversationID  uint
	UserID          uint
	RunID           string
	EventID         string
	EventType       string
	Phase           string
	Stage           string
	RoundID         string
	ParentEventID   string
	Status          string
	Title           string
	Summary         string
	ContentMarkdown string
	PayloadJSON     string
	Seq             int
	StartedAt       time.Time
	EndedAt         *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ToolCall 表示工具调用记录。
type ToolCall struct {
	ID             uint
	MessageID      uint
	ConversationID uint
	UserID         uint
	RunID          string
	ToolCallID     string
	ToolType       string
	ToolName       string
	Status         string
	LatencyMS      int64
	InputJSON      string
	OutputJSON     string
	ErrorJSON      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ContextSnapshot 表示上下文压缩快照。
type ContextSnapshot struct {
	ID             uint
	ConversationID uint
	MessageID      uint
	UserID         uint
	RunID          string
	FromTurn       int
	ToTurn         int
	SourceTokens   int64
	SummaryTokens  int64
	SummaryText    string
	Strategy       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RAGChunk 单个 RAG 检索到的文本片段及其来源信息。
type RAGChunk struct {
	Content    string
	FileName   string
	FileID     string
	ChunkIndex int
	Score      float32
}

// MessageChunk 表示消息向量分片，用于历史对话语义检索。
type MessageChunk struct {
	ID             uint
	ConversationID uint
	MessageID      uint
	UserID         uint
	Role           string
	ChunkIndex     int
	Content        string
	TokenCount     int
	Similarity     float64 // 检索时附加的相似度分数（写入时为 0）
	CreatedAt      time.Time
}

// ArenaVote 表示用户对一组竞技场分支的偏好投票。
type ArenaVote struct {
	ID             uint
	MessageGroupID string
	ConversationID uint
	VoterUserID    uint
	WinnerModel    string
	BlindMode      bool
	CreatedAt      time.Time
}

// ArenaLeaderboardEntry 表示偏好榜中单个模型的聚合统计。
type ArenaLeaderboardEntry struct {
	Model        string
	WinCount     int64
	TotalBattles int64
	WinRate      float64
}
