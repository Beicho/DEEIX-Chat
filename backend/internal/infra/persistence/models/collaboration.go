package model

import "time"

// Assistant stores a user-created assistant preset.
type Assistant struct {
	BaseModel
	PublicID       string     `gorm:"size:32;not null;default:'';uniqueIndex:idx_assistants_public_id;comment:公开助手ID"`
	OwnerUserID    uint       `gorm:"not null;index:idx_assistants_owner_user_id;comment:创建用户ID"`
	Name           string     `gorm:"size:80;not null;default:'';comment:助手名称"`
	AvatarURL      string     `gorm:"size:2048;not null;default:'';comment:头像地址"`
	Description    string     `gorm:"size:255;not null;default:'';comment:描述"`
	SystemPrompt   string     `gorm:"type:text;not null;default:'';comment:助手系统提示词"`
	DefaultModel   string     `gorm:"size:128;not null;default:'';comment:默认模型"`
	OpeningMessage string     `gorm:"type:text;not null;default:'';comment:开场白"`
	Visibility     string     `gorm:"size:32;not null;default:'private';index:idx_assistants_visibility;comment:可见范围(private/public)"`
	Status         string     `gorm:"size:32;not null;default:'active';index:idx_assistants_status;comment:状态(active/deleted)"`
	PublishedAt    *time.Time `gorm:"index:idx_assistants_published_at;comment:发布时间"`
}

func (Assistant) TableName() string { return "assistants" }

// AssistantInstall stores an assistant installation relationship.
type AssistantInstall struct {
	BaseModel
	UserID      uint `gorm:"not null;uniqueIndex:idx_assistant_installs_user_assistant,priority:1;index:idx_assistant_installs_user_id;comment:用户ID"`
	AssistantID uint `gorm:"not null;uniqueIndex:idx_assistant_installs_user_assistant,priority:2;index:idx_assistant_installs_assistant_id;comment:助手ID"`
}

func (AssistantInstall) TableName() string { return "assistant_installs" }

// ScheduledPrompt stores a scheduled prompt.
type ScheduledPrompt struct {
	BaseModel
	PublicID             string     `gorm:"size:32;not null;default:'';uniqueIndex:idx_scheduled_prompts_public_id;comment:公开定时提示ID"`
	UserID               uint       `gorm:"not null;index:idx_scheduled_prompts_user_id;comment:用户ID"`
	AssistantID          *uint      `gorm:"index:idx_scheduled_prompts_assistant_id;comment:助手ID"`
	TargetConversationID *uint      `gorm:"index:idx_scheduled_prompts_target_conversation_id;comment:目标会话ID"`
	Title                string     `gorm:"size:120;not null;default:'';comment:标题"`
	Content              string     `gorm:"type:text;not null;default:'';comment:提示内容"`
	DueAt                time.Time  `gorm:"not null;index:idx_scheduled_prompts_due_at;comment:兼容到期时间"`
	NextRunAt            time.Time  `gorm:"index:idx_scheduled_prompts_next_run_at;comment:下次运行时间"`
	ScheduleType         string     `gorm:"size:32;not null;default:'once';comment:计划类型(once/daily/weekly/cron)"`
	ScheduleTime         string     `gorm:"size:8;not null;default:'';comment:每天/每周运行时间HH:mm"`
	ScheduleWeekday      int        `gorm:"not null;default:0;comment:每周运行日"`
	CronExpression       string     `gorm:"size:128;not null;default:'';comment:cron表达式"`
	Model                string     `gorm:"size:128;not null;default:'';comment:运行模型"`
	Enabled              bool       `gorm:"not null;default:true;index:idx_scheduled_prompts_enabled;comment:是否启用"`
	Status               string     `gorm:"size:32;not null;default:'scheduled';index:idx_scheduled_prompts_status;comment:状态(scheduled/paused/deleted)"`
	LastTriggeredAt      *time.Time `gorm:"index:idx_scheduled_prompts_last_triggered_at;comment:最近触发时间"`
	RetryCount           int        `gorm:"not null;default:0;comment:当前运行重试次数"`
	LastError            string     `gorm:"type:text;not null;default:'';comment:最近错误"`
	ConversationID       *uint      `gorm:"index:idx_scheduled_prompts_conversation_id;comment:最近写入会话ID"`
}

func (ScheduledPrompt) TableName() string { return "scheduled_prompts" }

// TeamSpace stores a collaborative team space.
type TeamSpace struct {
	BaseModel
	PublicID    string `gorm:"size:32;not null;default:'';uniqueIndex:idx_team_spaces_public_id;comment:公开团队ID"`
	OwnerUserID uint   `gorm:"not null;index:idx_team_spaces_owner_user_id;comment:拥有者用户ID"`
	Name        string `gorm:"size:80;not null;default:'';comment:团队名称"`
	Description string `gorm:"size:255;not null;default:'';comment:描述"`
	Status      string `gorm:"size:32;not null;default:'active';index:idx_team_spaces_status;comment:状态(active/deleted)"`
}

func (TeamSpace) TableName() string { return "team_spaces" }

// TeamMember stores a team membership.
type TeamMember struct {
	BaseModel
	TeamID    uint   `gorm:"not null;uniqueIndex:idx_team_members_team_user,priority:1;index:idx_team_members_team_id;comment:团队ID"`
	UserID    uint   `gorm:"not null;uniqueIndex:idx_team_members_team_user,priority:2;index:idx_team_members_user_id;comment:用户ID"`
	Role      string `gorm:"size:32;not null;default:'member';index:idx_team_members_role;comment:角色(owner/admin/member)"`
	InvitedBy uint   `gorm:"not null;default:0;comment:邀请人用户ID"`
}

func (TeamMember) TableName() string { return "team_members" }

// ProjectDocument stores a project-level file reference.
type ProjectDocument struct {
	BaseModel
	UserID      uint   `gorm:"not null;index:idx_project_documents_user_id;uniqueIndex:idx_project_documents_project_file,priority:3;comment:用户ID"`
	ProjectID   uint   `gorm:"not null;index:idx_project_documents_project_id;uniqueIndex:idx_project_documents_project_file,priority:1;comment:项目ID"`
	FileObjID   uint   `gorm:"not null;index:idx_project_documents_file_obj_id;comment:文件对象主键ID"`
	FileID      string `gorm:"size:64;not null;default:'';uniqueIndex:idx_project_documents_project_file,priority:2;comment:文件对象ID"`
	IndexStatus string `gorm:"size:32;not null;default:'pending';index:idx_project_documents_index_status;comment:索引状态(pending/ready/failed/stale)"`
	Status      string `gorm:"size:32;not null;default:'active';index:idx_project_documents_status;comment:状态(active/deleted)"`
}

func (ProjectDocument) TableName() string { return "project_documents" }
