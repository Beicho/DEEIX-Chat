package conversation

import (
	"encoding/json"
	"time"

	appconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/conversation"
)

type conversationTakeoutPayload struct {
	Format             string                           `json:"format,omitempty"`
	Version            int                              `json:"version"`
	ExportScope        string                           `json:"exportScope,omitempty"`
	ExportedAt         time.Time                        `json:"exportedAt,omitempty"`
	Conversation       conversationTakeoutConversation  `json:"conversation,omitempty"`
	Messages           []conversationTakeoutMessage     `json:"messages,omitempty"`
	Conversations      []conversationTakeoutItem        `json:"conversations,omitempty"`
	TotalConversations int                              `json:"totalConversations,omitempty"`
	TotalMessages      int                              `json:"totalMessages,omitempty"`
	Compatibility      conversationTakeoutCompatibility `json:"compatibility,omitempty"`
}

type conversationTakeoutCompatibility struct {
	Format string `json:"format"`
	Notes  string `json:"notes"`
}

type conversationTakeoutItem struct {
	Conversation conversationTakeoutConversation `json:"conversation"`
	Messages     []conversationTakeoutMessage    `json:"messages"`
}

type conversationTakeoutConversation struct {
	PublicID   string          `json:"publicID,omitempty"`
	Title      string          `json:"title"`
	LabelsJSON string          `json:"labelsJSON,omitempty"`
	Labels     json.RawMessage `json:"labels,omitempty"`
	Model      string          `json:"model,omitempty"`
	Provider   string          `json:"provider,omitempty"`
	Status     string          `json:"status,omitempty"`
	CreatedAt  *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time      `json:"updatedAt,omitempty"`
}

type conversationTakeoutMessage struct {
	PublicID         string     `json:"publicID,omitempty"`
	ParentPublicID   string     `json:"parentPublicID,omitempty"`
	SourcePublicID   string     `json:"sourcePublicID,omitempty"`
	RunID            string     `json:"runID,omitempty"`
	Role             string     `json:"role"`
	ContentType      string     `json:"contentType"`
	Content          string     `json:"content"`
	BranchReason     string     `json:"branchReason,omitempty"`
	TokenUsage       int64      `json:"tokenUsage,omitempty"`
	InputTokens      int64      `json:"inputTokens,omitempty"`
	OutputTokens     int64      `json:"outputTokens,omitempty"`
	CacheReadTokens  int64      `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens int64      `json:"cacheWriteTokens,omitempty"`
	ReasoningTokens  int64      `json:"reasoningTokens,omitempty"`
	LatencyMS        int64      `json:"latencyMS,omitempty"`
	Status           string     `json:"status,omitempty"`
	ErrorCode        string     `json:"errorCode,omitempty"`
	ErrorMessage     string     `json:"errorMessage,omitempty"`
	Attachments      string     `json:"attachments,omitempty"`
	EditedAt         *time.Time `json:"editedAt,omitempty"`
	CreatedAt        *time.Time `json:"createdAt,omitempty"`
	UpdatedAt        *time.Time `json:"updatedAt,omitempty"`
}

func toConversationTakeoutPayload(item *appconversation.ConversationTakeout) conversationTakeoutPayload {
	if item == nil {
		return conversationTakeoutPayload{}
	}
	payload := conversationTakeoutPayload{
		Format: item.Format, Version: item.Version, ExportScope: item.ExportScope, ExportedAt: item.ExportedAt,
		Conversation:       toTakeoutConversationPayload(item.Conversation),
		Messages:           toTakeoutMessagePayloads(item.Messages),
		TotalConversations: item.TotalConversations, TotalMessages: item.TotalMessages,
		Compatibility: conversationTakeoutCompatibility{Format: item.Compatibility.Format, Notes: item.Compatibility.Notes},
		Conversations: make([]conversationTakeoutItem, 0, len(item.Conversations)),
	}
	for _, conversation := range item.Conversations {
		payload.Conversations = append(payload.Conversations, conversationTakeoutItem{
			Conversation: toTakeoutConversationPayload(conversation.Conversation),
			Messages:     toTakeoutMessagePayloads(conversation.Messages),
		})
	}
	return payload
}

func (payload conversationTakeoutPayload) toApplication() appconversation.ConversationTakeout {
	item := appconversation.ConversationTakeout{
		Format: payload.Format, Version: payload.Version, ExportScope: payload.ExportScope, ExportedAt: payload.ExportedAt,
		Conversation:       payload.Conversation.toApplication(),
		Messages:           takeoutMessagesToApplication(payload.Messages),
		TotalConversations: payload.TotalConversations, TotalMessages: payload.TotalMessages,
		Compatibility: appconversation.ConversationTakeoutCompatibility{Format: payload.Compatibility.Format, Notes: payload.Compatibility.Notes},
		Conversations: make([]appconversation.ConversationTakeoutItem, 0, len(payload.Conversations)),
	}
	for _, conversation := range payload.Conversations {
		item.Conversations = append(item.Conversations, appconversation.ConversationTakeoutItem{
			Conversation: conversation.Conversation.toApplication(),
			Messages:     takeoutMessagesToApplication(conversation.Messages),
		})
	}
	return item
}

func toTakeoutConversationPayload(item appconversation.ConversationTakeoutConversation) conversationTakeoutConversation {
	return conversationTakeoutConversation{
		PublicID: item.PublicID, Title: item.Title, LabelsJSON: item.LabelsJSON, Labels: item.Labels,
		Model: item.Model, Provider: item.Provider, Status: item.Status, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func (item conversationTakeoutConversation) toApplication() appconversation.ConversationTakeoutConversation {
	return appconversation.ConversationTakeoutConversation{
		PublicID: item.PublicID, Title: item.Title, LabelsJSON: item.LabelsJSON, Labels: item.Labels,
		Model: item.Model, Provider: item.Provider, Status: item.Status, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func toTakeoutMessagePayloads(items []appconversation.ConversationTakeoutMessage) []conversationTakeoutMessage {
	result := make([]conversationTakeoutMessage, 0, len(items))
	for _, item := range items {
		result = append(result, conversationTakeoutMessage{
			PublicID: item.PublicID, ParentPublicID: item.ParentPublicID, SourcePublicID: item.SourcePublicID, RunID: item.RunID,
			Role: item.Role, ContentType: item.ContentType, Content: item.Content, BranchReason: item.BranchReason,
			TokenUsage: item.TokenUsage, InputTokens: item.InputTokens, OutputTokens: item.OutputTokens,
			CacheReadTokens: item.CacheReadTokens, CacheWriteTokens: item.CacheWriteTokens, ReasoningTokens: item.ReasoningTokens,
			LatencyMS: item.LatencyMS, Status: item.Status, ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
			Attachments: item.Attachments, EditedAt: item.EditedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return result
}

func takeoutMessagesToApplication(items []conversationTakeoutMessage) []appconversation.ConversationTakeoutMessage {
	result := make([]appconversation.ConversationTakeoutMessage, 0, len(items))
	for _, item := range items {
		result = append(result, appconversation.ConversationTakeoutMessage{
			PublicID: item.PublicID, ParentPublicID: item.ParentPublicID, SourcePublicID: item.SourcePublicID, RunID: item.RunID,
			Role: item.Role, ContentType: item.ContentType, Content: item.Content, BranchReason: item.BranchReason,
			TokenUsage: item.TokenUsage, InputTokens: item.InputTokens, OutputTokens: item.OutputTokens,
			CacheReadTokens: item.CacheReadTokens, CacheWriteTokens: item.CacheWriteTokens, ReasoningTokens: item.ReasoningTokens,
			LatencyMS: item.LatencyMS, Status: item.Status, ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
			Attachments: item.Attachments, EditedAt: item.EditedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return result
}
