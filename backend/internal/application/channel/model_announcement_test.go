package channel

import (
	"context"
	"testing"

	domainannouncement "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/announcement"
)

func TestNotifyPlatformModelCreatedCreatesNewModelDraft(t *testing.T) {
	recorder := &recordingModelAnnouncementService{}
	service := &Service{modelAnnouncementService: recorder}

	service.notifyPlatformModelCreated(context.Background(), "gpt-5.4-mini")

	if len(recorder.modelNames) != 1 {
		t.Fatalf("created drafts = %d, want 1", len(recorder.modelNames))
	}
	if recorder.modelNames[0] != "gpt-5.4-mini" {
		t.Fatalf("model name = %q", recorder.modelNames[0])
	}
}

func TestNewModelAnnouncementDraftInputUsesDraftCopy(t *testing.T) {
	got, ok := newModelAnnouncementDraftInput("gpt-5.4-mini")
	if !ok {
		t.Fatal("expected draft input")
	}
	if got.Title != "新模型上线：gpt-5.4-mini" {
		t.Fatalf("title = %q", got.Title)
	}
	if got.Status != domainannouncement.StatusDraft {
		t.Fatalf("status = %q, want draft", got.Status)
	}
	if got.Type != domainannouncement.TypeInfo {
		t.Fatalf("type = %q, want info", got.Type)
	}
	if got.ContentMarkdown != "gpt-5.4-mini 已加入模型列表。发布后，用户可在公告中看到这条说明。" {
		t.Fatalf("content = %q", got.ContentMarkdown)
	}
}

func TestNotifyPlatformModelCreatedSkipsEmptyModelName(t *testing.T) {
	recorder := &recordingModelAnnouncementService{}
	service := &Service{modelAnnouncementService: recorder}

	service.notifyPlatformModelCreated(context.Background(), " ")

	if len(recorder.modelNames) != 0 {
		t.Fatalf("created drafts = %d, want 0", len(recorder.modelNames))
	}
}

type recordingModelAnnouncementService struct {
	modelNames []string
}

func (s *recordingModelAnnouncementService) CreateNewModelDraft(_ context.Context, modelName string) error {
	s.modelNames = append(s.modelNames, modelName)
	return nil
}
