package channel

import (
	"context"
	"strings"

	domainannouncement "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/announcement"
	"go.uber.org/zap"
)

// NewModelAnnouncementDraftInput 是渠道模块发起的新模型公告草稿请求。
type NewModelAnnouncementDraftInput struct {
	Title           string
	ContentMarkdown string
	Status          string
	Type            string
}

type modelAnnouncementService interface {
	CreateNewModelDraft(ctx context.Context, modelName string) error
}

// SetModelAnnouncementService 注入公告服务，用于新平台模型自动生成草稿公告。
func (s *Service) SetModelAnnouncementService(service modelAnnouncementService) {
	s.modelAnnouncementService = service
}

func (s *Service) notifyPlatformModelCreated(ctx context.Context, modelName string) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" || s == nil || s.modelAnnouncementService == nil {
		return
	}
	if err := s.modelAnnouncementService.CreateNewModelDraft(ctx, modelName); err != nil {
		s.warn("create_new_model_announcement_draft_failed", zap.String("model", modelName), zap.Error(err))
	}
}

func newModelAnnouncementDraftInput(modelName string) (NewModelAnnouncementDraftInput, bool) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return NewModelAnnouncementDraftInput{}, false
	}
	return NewModelAnnouncementDraftInput{
		Title:           "新模型上线：" + modelName,
		ContentMarkdown: modelName + " 已加入模型列表。发布后，用户可在公告中看到这条说明。",
		Status:          domainannouncement.StatusDraft,
		Type:            domainannouncement.TypeInfo,
	}, true
}
