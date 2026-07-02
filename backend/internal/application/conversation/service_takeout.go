package conversation

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	"github.com/google/uuid"
)

const (
	conversationTakeoutFormat       = "deeix.conversations.takeout"
	maxTakeoutImportConversations   = 100
	maxTakeoutImportMessages        = 5000
	maxTakeoutImportTitleRunes      = 255
	maxTakeoutImportContentBytes    = 2 * 1024 * 1024
	maxTakeoutImportAttachmentsJSON = 512 * 1024
)

// ConversationTakeout is the portable JSON shape for user-owned conversation data.
type ConversationTakeout struct {
	Format             string                           `json:"format,omitempty"`
	Version            int                              `json:"version"`
	ExportScope        string                           `json:"exportScope,omitempty"`
	ExportedAt         time.Time                        `json:"exportedAt,omitempty"`
	Conversation       ConversationTakeoutConversation  `json:"conversation,omitempty"`
	Messages           []ConversationTakeoutMessage     `json:"messages,omitempty"`
	Conversations      []ConversationTakeoutItem        `json:"conversations,omitempty"`
	TotalConversations int                              `json:"totalConversations,omitempty"`
	TotalMessages      int                              `json:"totalMessages,omitempty"`
	Compatibility      ConversationTakeoutCompatibility `json:"compatibility,omitempty"`
}

type ConversationTakeoutCompatibility struct {
	Format string `json:"format"`
	Notes  string `json:"notes"`
}

type ConversationTakeoutItem struct {
	Conversation ConversationTakeoutConversation `json:"conversation"`
	Messages     []ConversationTakeoutMessage    `json:"messages"`
}

type ConversationTakeoutConversation struct {
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

type ConversationTakeoutMessage struct {
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

type ConversationImportResult struct {
	ImportedConversationCount int                  `json:"importedConversationCount"`
	ImportedMessageCount      int                  `json:"importedMessageCount"`
	Conversations             []model.Conversation `json:"conversations"`
}

// ExportConversationTakeout exports all visible conversations owned by the current user.
func (s *Service) ExportConversationTakeout(ctx context.Context, userID uint) (*ConversationTakeout, error) {
	items, total, err := s.repo.ListConversationsByUser(ctx, userID, 0, maxTakeoutImportConversations, "all", "all", "all", "all", "")
	if err != nil {
		return nil, err
	}
	takeout := &ConversationTakeout{
		Format:             conversationTakeoutFormat,
		Version:            conversationExportVersion,
		ExportScope:        conversationExportScopeFull,
		ExportedAt:         time.Now().UTC(),
		Conversations:      make([]ConversationTakeoutItem, 0, len(items)),
		TotalConversations: int(total),
		Compatibility: ConversationTakeoutCompatibility{
			Format: conversationTakeoutFormat,
			Notes:  "User conversation data export.",
		},
	}
	for index := range items {
		messages, err := s.repo.ListAllMessages(ctx, items[index].ID)
		if err != nil {
			return nil, err
		}
		takeout.Conversations = append(takeout.Conversations, ConversationTakeoutItem{
			Conversation: takeoutConversationFromDomain(items[index]),
			Messages:     takeoutMessagesFromDomain(messages),
		})
		takeout.TotalMessages += len(messages)
	}
	return takeout, nil
}

// ImportConversationTakeout validates portable JSON and creates new conversations for the current user.
func (s *Service) ImportConversationTakeout(ctx context.Context, userID uint, input ConversationTakeout) (*ConversationImportResult, error) {
	takeout := normalizeConversationTakeoutInput(input)
	if err := validateConversationTakeout(&takeout); err != nil {
		return nil, err
	}
	result := &ConversationImportResult{
		Conversations: make([]model.Conversation, 0, len(takeout.Conversations)),
	}
	for _, item := range takeout.Conversations {
		conversation := &model.Conversation{
			UserID:        userID,
			PublicID:      normalizePublicID(uuid.NewString()),
			Title:         normalizeTakeoutTitle(item.Conversation.Title),
			LabelsJSON:    normalizeTakeoutLabelsJSON(item.Conversation),
			Model:         strings.TrimSpace(item.Conversation.Model),
			Provider:      strings.TrimSpace(item.Conversation.Provider),
			SessionKey:    uuid.NewString(),
			MessageCount:  0,
			Status:        "active",
			ContextPolicy: buildContextPolicyJSON(s.cfg.Snapshot()),
		}
		if conversation.Provider == "" {
			conversation.Provider = inferProvider(conversation.Model)
		}
		if err := s.repo.CreateConversation(ctx, conversation); err != nil {
			return nil, err
		}
		idMap := make(map[string]uint, len(item.Messages))
		for _, exported := range item.Messages {
			message := messageFromTakeout(userID, conversation.ID, exported, idMap)
			if err := s.repo.CreateMessage(ctx, message); err != nil {
				return nil, err
			}
			if strings.TrimSpace(exported.PublicID) != "" {
				idMap[strings.TrimSpace(exported.PublicID)] = message.ID
			}
			result.ImportedMessageCount++
		}
		if err := s.repo.IncrementMessageCount(ctx, conversation.ID, len(item.Messages)); err != nil {
			return nil, err
		}
		conversation.MessageCount = len(item.Messages)
		result.ImportedConversationCount++
		result.Conversations = append(result.Conversations, *conversation)
	}
	return result, nil
}

func normalizeConversationTakeoutInput(input ConversationTakeout) ConversationTakeout {
	if len(input.Conversations) == 0 && (strings.TrimSpace(input.Conversation.Title) != "" || len(input.Messages) > 0) {
		input.Conversations = []ConversationTakeoutItem{{
			Conversation: input.Conversation,
			Messages:     input.Messages,
		}}
	}
	return input
}

func validateConversationTakeout(input *ConversationTakeout) error {
	if input == nil {
		return ErrInvalidConversationImport
	}
	if input.Version != conversationExportVersion {
		return ErrInvalidConversationImport
	}
	format := strings.TrimSpace(input.Format)
	if format != "" && format != conversationTakeoutFormat {
		return ErrInvalidConversationImport
	}
	if len(input.Conversations) == 0 || len(input.Conversations) > maxTakeoutImportConversations {
		return ErrInvalidConversationImport
	}
	totalMessages := 0
	for _, item := range input.Conversations {
		if strings.TrimSpace(item.Conversation.Title) == "" {
			return ErrInvalidConversationImport
		}
		if len([]rune(item.Conversation.Title)) > maxTakeoutImportTitleRunes {
			return ErrInvalidConversationImport
		}
		labelsJSON := normalizeTakeoutLabelsJSON(item.Conversation)
		if !json.Valid([]byte(labelsJSON)) {
			return ErrInvalidConversationImport
		}
		totalMessages += len(item.Messages)
		if totalMessages > maxTakeoutImportMessages {
			return ErrInvalidConversationImport
		}
		for _, message := range item.Messages {
			if err := validateTakeoutMessage(message); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTakeoutMessage(message ConversationTakeoutMessage) error {
	switch strings.TrimSpace(message.Role) {
	case "user", "assistant", "system", "tool":
	default:
		return ErrInvalidConversationImport
	}
	contentType := strings.TrimSpace(message.ContentType)
	if contentType == "" {
		contentType = "text"
	}
	switch contentType {
	case "text", "markdown", "image", "file", "mixed":
	default:
		return ErrInvalidConversationImport
	}
	if len([]byte(message.Content)) > maxTakeoutImportContentBytes {
		return ErrInvalidConversationImport
	}
	attachments := strings.TrimSpace(message.Attachments)
	if attachments == "" {
		return nil
	}
	if len([]byte(attachments)) > maxTakeoutImportAttachmentsJSON || !json.Valid([]byte(attachments)) {
		return ErrInvalidConversationImport
	}
	return nil
}

func takeoutConversationFromDomain(item model.Conversation) ConversationTakeoutConversation {
	createdAt := item.CreatedAt
	updatedAt := item.UpdatedAt
	return ConversationTakeoutConversation{
		PublicID:   item.PublicID,
		Title:      item.Title,
		LabelsJSON: normalizeRawJSONArray(item.LabelsJSON),
		Model:      item.Model,
		Provider:   item.Provider,
		Status:     item.Status,
		CreatedAt:  &createdAt,
		UpdatedAt:  &updatedAt,
	}
}

func takeoutMessagesFromDomain(items []model.Message) []ConversationTakeoutMessage {
	results := make([]ConversationTakeoutMessage, 0, len(items))
	for _, item := range items {
		createdAt := item.CreatedAt
		updatedAt := item.UpdatedAt
		results = append(results, ConversationTakeoutMessage{
			PublicID:         item.PublicID,
			ParentPublicID:   item.ParentPublicID,
			SourcePublicID:   item.SourcePublicID,
			RunID:            item.RunID,
			Role:             item.Role,
			ContentType:      item.ContentType,
			Content:          item.Content,
			BranchReason:     item.BranchReason,
			TokenUsage:       item.TokenUsage,
			InputTokens:      item.InputTokens,
			OutputTokens:     item.OutputTokens,
			CacheReadTokens:  item.CacheReadTokens,
			CacheWriteTokens: item.CacheWriteTokens,
			ReasoningTokens:  item.ReasoningTokens,
			LatencyMS:        item.LatencyMS,
			Status:           item.Status,
			ErrorCode:        item.ErrorCode,
			ErrorMessage:     item.ErrorMessage,
			Attachments:      normalizeRawJSONArray(item.Attachments),
			EditedAt:         item.EditedAt,
			CreatedAt:        &createdAt,
			UpdatedAt:        &updatedAt,
		})
	}
	return results
}

func messageFromTakeout(userID uint, conversationID uint, item ConversationTakeoutMessage, idMap map[string]uint) *model.Message {
	var parentID *uint
	if id, ok := idMap[strings.TrimSpace(item.ParentPublicID)]; ok {
		parentID = &id
	}
	var sourceID *uint
	if id, ok := idMap[strings.TrimSpace(item.SourcePublicID)]; ok {
		sourceID = &id
	}
	contentType := strings.TrimSpace(item.ContentType)
	if contentType == "" {
		contentType = "text"
	}
	branchReason := strings.TrimSpace(item.BranchReason)
	if branchReason == "" {
		branchReason = "default"
	}
	status := strings.TrimSpace(item.Status)
	if status == "" {
		status = "completed"
	}
	return &model.Message{
		ConversationID:   conversationID,
		UserID:           userID,
		PublicID:         normalizePublicID(uuid.NewString()),
		ParentMessageID:  parentID,
		RunID:            "",
		Role:             strings.TrimSpace(item.Role),
		ContentType:      contentType,
		Content:          item.Content,
		BranchReason:     branchReason,
		SourceMessageID:  sourceID,
		TokenUsage:       item.TokenUsage,
		InputTokens:      item.InputTokens,
		OutputTokens:     item.OutputTokens,
		CacheReadTokens:  item.CacheReadTokens,
		CacheWriteTokens: item.CacheWriteTokens,
		ReasoningTokens:  item.ReasoningTokens,
		LatencyMS:        item.LatencyMS,
		BilledCurrency:   "USD",
		BilledNanousd:    0,
		PricingSnapshot:  "",
		Status:           status,
		ErrorCode:        item.ErrorCode,
		ErrorMessage:     item.ErrorMessage,
		EditedAt:         item.EditedAt,
	}
}

func normalizeTakeoutTitle(value string) string {
	title := strings.TrimSpace(value)
	if title == "" {
		return "Imported conversation"
	}
	runes := []rune(title)
	if len(runes) > maxTakeoutImportTitleRunes {
		return string(runes[:maxTakeoutImportTitleRunes])
	}
	return title
}

func normalizeTakeoutLabelsJSON(item ConversationTakeoutConversation) string {
	if len(item.Labels) > 0 && json.Valid(item.Labels) {
		return normalizeRawJSONArray(string(item.Labels))
	}
	return normalizeRawJSONArray(item.LabelsJSON)
}

func normalizeRawJSONArray(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" || normalized == "null" {
		return "[]"
	}
	var raw []json.RawMessage
	if err := json.Unmarshal([]byte(normalized), &raw); err != nil {
		return "[]"
	}
	compacted, err := json.Marshal(raw)
	if err != nil {
		return "[]"
	}
	return string(compacted)
}
