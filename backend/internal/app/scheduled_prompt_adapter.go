package app

import (
	"context"

	appcollaboration "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/collaboration"
	appconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/conversation"
	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
)

type scheduledPromptConversationAdapter struct {
	service *appconversation.Service
}

func (a scheduledPromptConversationAdapter) CreateConversation(ctx context.Context, userID uint, title string, modelName string, projectPublicID string) (*domainconversation.Conversation, error) {
	return a.service.CreateConversation(ctx, userID, title, modelName, projectPublicID)
}

func (a scheduledPromptConversationAdapter) SendScheduledPromptMessage(ctx context.Context, input appcollaboration.ScheduledPromptSendInput) (*appcollaboration.ScheduledPromptSendResult, error) {
	result, err := a.service.SendMessage(ctx, appconversation.SendMessageInput{
		UserID:            input.UserID,
		ConversationID:    input.ConversationID,
		RequestID:         input.RequestID,
		ContentType:       "text",
		Content:           input.Content,
		PlatformModelName: input.PlatformModelName,
		AssistantPublicID: input.AssistantPublicID,
	})
	if err != nil {
		return nil, err
	}
	return &appcollaboration.ScheduledPromptSendResult{AssistantContent: result.AssistantMessage.Content}, nil
}
