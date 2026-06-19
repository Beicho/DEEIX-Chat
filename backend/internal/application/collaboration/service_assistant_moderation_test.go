package collaboration

import (
	"context"
	"errors"
	"testing"
	"time"

	domaincollab "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/collaboration"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

type assistantModerationRepo struct {
	created *domaincollab.Assistant
	updated bool
}

func (r *assistantModerationRepo) CreateAssistant(ctx context.Context, item *domaincollab.Assistant) error {
	r.created = item
	return nil
}

func (r *assistantModerationRepo) ListAssistants(ctx context.Context, userID uint, includePublic bool, offset int, limit int) ([]domaincollab.Assistant, int64, error) {
	return nil, 0, nil
}

func (r *assistantModerationRepo) ListPublicAssistants(ctx context.Context, userID uint, offset int, limit int) ([]domaincollab.Assistant, int64, error) {
	return nil, 0, nil
}

func (r *assistantModerationRepo) GetAssistantByPublicID(ctx context.Context, userID uint, publicID string) (*domaincollab.Assistant, error) {
	return nil, repository.ErrNotFound
}

func (r *assistantModerationRepo) UpdateAssistant(ctx context.Context, userID uint, publicID string, patch repository.AssistantPatch) (*domaincollab.Assistant, error) {
	r.updated = true
	return &domaincollab.Assistant{PublicID: publicID, OwnerUserID: userID, Visibility: valueOrEmpty(patch.Visibility)}, nil
}

func (r *assistantModerationRepo) DeleteAssistant(ctx context.Context, userID uint, publicID string) error {
	return nil
}

func (r *assistantModerationRepo) InstallAssistant(ctx context.Context, userID uint, assistantPublicID string) (*domaincollab.AssistantInstall, error) {
	return nil, nil
}

func (r *assistantModerationRepo) UninstallAssistant(ctx context.Context, userID uint, assistantPublicID string) error {
	return nil
}

func (r *assistantModerationRepo) GetInstalledAssistantPrompt(ctx context.Context, userID uint, publicID string) (*domaincollab.Assistant, error) {
	return nil, repository.ErrNotFound
}

func (r *assistantModerationRepo) CreateScheduledPrompt(ctx context.Context, item *domaincollab.ScheduledPrompt) error {
	return nil
}

func (r *assistantModerationRepo) ListScheduledPrompts(ctx context.Context, userID uint, offset int, limit int) ([]domaincollab.ScheduledPrompt, int64, error) {
	return nil, 0, nil
}

func (r *assistantModerationRepo) UpdateScheduledPrompt(ctx context.Context, userID uint, publicID string, patch repository.ScheduledPromptPatch) (*domaincollab.ScheduledPrompt, error) {
	return nil, repository.ErrNotFound
}

func (r *assistantModerationRepo) DeleteScheduledPrompt(ctx context.Context, userID uint, publicID string) error {
	return nil
}

func (r *assistantModerationRepo) ListDueScheduledPrompts(ctx context.Context, now time.Time, limit int) ([]domaincollab.ScheduledPrompt, error) {
	return nil, nil
}

func (r *assistantModerationRepo) GetConversationTarget(ctx context.Context, userID uint, publicID string) (uint, string, error) {
	return 0, "", repository.ErrNotFound
}

func (r *assistantModerationRepo) MarkScheduledPromptSucceeded(ctx context.Context, id uint, now time.Time, nextRunAt *time.Time, conversationID uint) error {
	return nil
}

func (r *assistantModerationRepo) MarkScheduledPromptFailed(ctx context.Context, id uint, lastError string, retryAt *time.Time, disable bool) error {
	return nil
}

func (r *assistantModerationRepo) CreateTeamSpace(ctx context.Context, team *domaincollab.TeamSpace, owner *domaincollab.TeamMember) error {
	return nil
}

func (r *assistantModerationRepo) ListTeamSpaces(ctx context.Context, userID uint) ([]domaincollab.TeamSpace, error) {
	return nil, nil
}

func (r *assistantModerationRepo) GetTeamSpaceByPublicID(ctx context.Context, userID uint, publicID string) (*domaincollab.TeamSpace, error) {
	return nil, repository.ErrNotFound
}

func (r *assistantModerationRepo) AddTeamMember(ctx context.Context, teamPublicID string, actorUserID uint, memberUserID uint, role string) (*domaincollab.TeamMember, error) {
	return nil, nil
}

func (r *assistantModerationRepo) RemoveTeamMember(ctx context.Context, teamPublicID string, actorUserID uint, memberUserID uint) error {
	return nil
}

func (r *assistantModerationRepo) FindUserByLogin(ctx context.Context, login string) (*domainuser.User, error) {
	return nil, repository.ErrNotFound
}

type assistantPolicyChecker struct {
	calls     int
	lastText  string
	direction string
	err       error
}

func (c *assistantPolicyChecker) CheckContentForPolicy(ctx context.Context, userID uint, content string, direction string) error {
	c.calls++
	c.lastText = content
	c.direction = direction
	return c.err
}

func TestCreatePublicAssistantRunsContentPolicyBeforePersist(t *testing.T) {
	repo := &assistantModerationRepo{}
	checker := &assistantPolicyChecker{err: errors.New("blocked")}
	service := NewService(repo, nil)
	service.SetContentPolicyChecker(checker)

	_, err := service.CreateAssistant(context.Background(), 42, AssistantInput{
		Name:           "public helper",
		Description:    "desc",
		SystemPrompt:   "unsafe prompt",
		OpeningMessage: "hello",
		Visibility:     domaincollab.AssistantVisibilityPublic,
	})

	if !errors.Is(err, ErrContentBlocked) {
		t.Fatalf("err = %v, want ErrContentBlocked", err)
	}
	if checker.calls != 1 {
		t.Fatalf("policy calls = %d, want 1", checker.calls)
	}
	if checker.direction != "assistant_publish" {
		t.Fatalf("direction = %q, want assistant_publish", checker.direction)
	}
	if repo.created != nil {
		t.Fatalf("assistant was persisted before policy passed")
	}
}

func TestCreatePrivateAssistantSkipsContentPolicy(t *testing.T) {
	repo := &assistantModerationRepo{}
	checker := &assistantPolicyChecker{err: errors.New("blocked")}
	service := NewService(repo, nil)
	service.SetContentPolicyChecker(checker)

	_, err := service.CreateAssistant(context.Background(), 42, AssistantInput{
		Name:         "private helper",
		SystemPrompt: "normal prompt",
		Visibility:   domaincollab.AssistantVisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("CreateAssistant() error = %v", err)
	}
	if checker.calls != 0 {
		t.Fatalf("policy calls = %d, want 0", checker.calls)
	}
	if repo.created == nil {
		t.Fatalf("assistant was not persisted")
	}
}

func TestUpdatePublicAssistantRunsContentPolicyBeforePersist(t *testing.T) {
	repo := &assistantModerationRepo{}
	checker := &assistantPolicyChecker{err: errors.New("blocked")}
	service := NewService(repo, nil)
	service.SetContentPolicyChecker(checker)

	_, err := service.UpdateAssistant(context.Background(), 42, "assistant_1", AssistantInput{
		Name:         "public helper",
		SystemPrompt: "unsafe prompt",
		Visibility:   domaincollab.AssistantVisibilityPublic,
	})

	if !errors.Is(err, ErrContentBlocked) {
		t.Fatalf("err = %v, want ErrContentBlocked", err)
	}
	if checker.calls != 1 {
		t.Fatalf("policy calls = %d, want 1", checker.calls)
	}
	if checker.direction != "assistant_publish" {
		t.Fatalf("direction = %q, want assistant_publish", checker.direction)
	}
	if repo.updated {
		t.Fatal("assistant update was persisted before policy passed")
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
