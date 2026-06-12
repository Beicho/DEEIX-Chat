package collaboration

import "time"

type AssistantRequest struct {
	Name           string `json:"name" binding:"required,max=80"`
	AvatarURL      string `json:"avatarURL" binding:"omitempty,max=2048"`
	Description    string `json:"description" binding:"omitempty,max=255"`
	SystemPrompt   string `json:"systemPrompt" binding:"required,max=12000"`
	DefaultModel   string `json:"defaultModel" binding:"omitempty,max=128"`
	OpeningMessage string `json:"openingMessage" binding:"omitempty,max=2000"`
	Visibility     string `json:"visibility" binding:"omitempty,oneof=private public"`
}

type ScheduledPromptRequest struct {
	AssistantID          string    `json:"assistantID" binding:"omitempty,max=32"`
	TargetConversationID string    `json:"targetConversationID" binding:"omitempty,max=32"`
	Title                string    `json:"title" binding:"required,max=120"`
	Content              string    `json:"content" binding:"required,max=20000"`
	DueAt                time.Time `json:"dueAt"`
	ScheduleType         string    `json:"scheduleType" binding:"omitempty,oneof=once daily weekly cron"`
	ScheduleTime         string    `json:"scheduleTime" binding:"omitempty,max=8"`
	ScheduleWeekday      int       `json:"scheduleWeekday"`
	CronExpression       string    `json:"cronExpression" binding:"omitempty,max=128"`
	Model                string    `json:"model" binding:"omitempty,max=128"`
	Enabled              bool      `json:"enabled"`
}

type TeamSpaceRequest struct {
	Name        string `json:"name" binding:"required,max=80"`
	Description string `json:"description" binding:"omitempty,max=255"`
}

type TeamMemberRequest struct {
	Login string `json:"login" binding:"required,max=128"`
	Role  string `json:"role" binding:"omitempty,oneof=admin member"`
}

type AssistantResponse struct {
	PublicID       string     `json:"publicID"`
	OwnerUserID    uint       `json:"ownerUserID"`
	Name           string     `json:"name"`
	AvatarURL      string     `json:"avatarURL"`
	Description    string     `json:"description"`
	SystemPrompt   string     `json:"systemPrompt"`
	DefaultModel   string     `json:"defaultModel"`
	OpeningMessage string     `json:"openingMessage"`
	Visibility     string     `json:"visibility"`
	Status         string     `json:"status"`
	PublishedAt    *time.Time `json:"publishedAt"`
	Installed      bool       `json:"installed"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type ScheduledPromptResponse struct {
	PublicID                string     `json:"publicID"`
	TargetConversationID    string     `json:"targetConversationID"`
	TargetConversationTitle string     `json:"targetConversationTitle"`
	Title                   string     `json:"title"`
	Content                 string     `json:"content"`
	DueAt                   time.Time  `json:"dueAt"`
	NextRunAt               time.Time  `json:"nextRunAt"`
	ScheduleType            string     `json:"scheduleType"`
	ScheduleTime            string     `json:"scheduleTime"`
	ScheduleWeekday         int        `json:"scheduleWeekday"`
	CronExpression          string     `json:"cronExpression"`
	Model                   string     `json:"model"`
	Enabled                 bool       `json:"enabled"`
	Status                  string     `json:"status"`
	LastTriggeredAt         *time.Time `json:"lastTriggeredAt"`
	RetryCount              int        `json:"retryCount"`
	LastError               string     `json:"lastError"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

type TeamSpaceResponse struct {
	PublicID    string               `json:"publicID"`
	OwnerUserID uint                 `json:"ownerUserID"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      string               `json:"status"`
	Members     []TeamMemberResponse `json:"members"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}

type TeamMemberResponse struct {
	UserID      uint      `json:"userID"`
	Role        string    `json:"role"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   string    `json:"avatarURL"`
	Email       string    `json:"email"`
	CreatedAt   time.Time `json:"createdAt"`
}
