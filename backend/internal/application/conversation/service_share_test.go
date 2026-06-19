package conversation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

func TestBuildShareMetadataDescriptionUsesFirstUserMessage(t *testing.T) {
	messages := []model.Message{
		{ID: 1, Role: "system", Content: "hidden"},
		{ID: 2, Role: "user", Content: "# Hello\n\nThis is **visible** text."},
		{ID: 3, Role: "assistant", Content: "assistant reply"},
	}

	got := buildShareMetadataDescription(messages, "fallback")
	want := "Hello This is visible text."
	if got != want {
		t.Fatalf("description mismatch: got %q, want %q", got, want)
	}
}

func TestBuildShareMetadataDescriptionTruncatesLongAssistantFallback(t *testing.T) {
	got := buildShareMetadataDescription([]model.Message{{Role: "assistant", Content: strings.Repeat("x", 220)}}, "fallback")
	if len([]rune(got)) != 160 {
		t.Fatalf("expected 160 runes, got %d: %q", len([]rune(got)), got)
	}
}

func TestSharedMessagesIncludeFileUsesSnapshotAttachments(t *testing.T) {
	messages := []model.Message{
		{ID: 1, PublicID: "u1", Role: "user", Attachments: `[{"file_id":"file_a"}]`},
		{ID: 2, PublicID: "a1", ParentPublicID: "u1", Role: "assistant", Attachments: `[{"file_id":"file_b"}]`},
	}

	if !sharedMessagesIncludeFile(messages, "file_a") {
		t.Fatal("expected file_a to be included")
	}
	if !sharedMessagesIncludeFile(messages, "file_b") {
		t.Fatal("expected file_b to be included")
	}
	if sharedMessagesIncludeFile(messages, "file_c") {
		t.Fatal("did not expect file_c to be included")
	}
}

func TestResolvePublicDefaultMessageIDsUsesStoredPath(t *testing.T) {
	messages := []model.Message{
		{ID: 1, PublicID: "u1", Role: "user"},
		{ID: 2, PublicID: "a1", ParentPublicID: "u1", Role: "assistant"},
		{ID: 3, PublicID: "u2-old", ParentPublicID: "a1", Role: "user"},
		{ID: 4, PublicID: "a2-old", ParentPublicID: "u2-old", Role: "assistant"},
		{ID: 5, PublicID: "u2-new", ParentPublicID: "a1", Role: "user"},
		{ID: 6, PublicID: "a2-new", ParentPublicID: "u2-new", Role: "assistant"},
	}

	got := resolvePublicDefaultMessageIDs(`["u1","a1","u2-old","a2-old"]`, messages)
	want := []string{"u1", "a1", "u2-old", "a2-old"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default branch mismatch: got %v, want %v", got, want)
	}
}

func TestResolvePublicDefaultMessageIDsFallsBackToLatestBranch(t *testing.T) {
	messages := []model.Message{
		{ID: 1, PublicID: "u1", Role: "user"},
		{ID: 2, PublicID: "a1", ParentPublicID: "u1", Role: "assistant"},
		{ID: 3, PublicID: "u2-old", ParentPublicID: "a1", Role: "user"},
		{ID: 4, PublicID: "a2-old", ParentPublicID: "u2-old", Role: "assistant"},
		{ID: 5, PublicID: "u2-new", ParentPublicID: "a1", Role: "user"},
		{ID: 6, PublicID: "a2-new", ParentPublicID: "u2-new", Role: "assistant"},
	}

	got := resolvePublicDefaultMessageIDs("", messages)
	want := []string{"u1", "a1", "u2-new", "a2-new"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback branch mismatch: got %v, want %v", got, want)
	}
}

func TestOrderSharedMessagesForCloneMakesDefaultBranchLatest(t *testing.T) {
	messages := []model.Message{
		{ID: 1, PublicID: "u1", Role: "user"},
		{ID: 2, PublicID: "a1", ParentPublicID: "u1", Role: "assistant"},
		{ID: 3, PublicID: "u2-old", ParentPublicID: "a1", Role: "user"},
		{ID: 4, PublicID: "a2-old", ParentPublicID: "u2-old", Role: "assistant"},
		{ID: 5, PublicID: "u2-new", ParentPublicID: "a1", Role: "user"},
		{ID: 6, PublicID: "a2-new", ParentPublicID: "u2-new", Role: "assistant"},
	}

	ordered := orderSharedMessagesForClone(messages, []string{"u1", "a1", "u2-old", "a2-old"})
	got := make([]string, 0, len(ordered))
	for _, message := range ordered {
		got = append(got, message.PublicID)
	}
	want := []string{"u1", "a1", "u2-new", "a2-new", "u2-old", "a2-old"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("clone order mismatch: got %v, want %v", got, want)
	}
}

func TestSanitizeSharedTracePayloadJSONRemovesInternalFields(t *testing.T) {
	got := sanitizeSharedTracePayloadJSON(`{
		"tool_calls": [{"tool_call_id":"call_1","output":"ok"}],
		"upstream_debug": {"authorization":"Bearer token"},
		"upstream": {"name":"hidden","model":"visible"},
		"api_key": "secret"
	}`)
	want := `{"tool_calls":[{"output":"ok","tool_call_id":"call_1"}],"upstream":{"model":"visible"}}`
	if got != want {
		t.Fatalf("sanitized payload mismatch: got %s, want %s", got, want)
	}
}

func TestNormalizeMessagePublicIDsDeduplicatesAndKeepsOrder(t *testing.T) {
	got := normalizeMessagePublicIDs([]string{"", " msg_a ", "msg_b", "msg_a", "\n"})
	want := []string{"msg_a", "msg_b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized ids mismatch: got %v, want %v", got, want)
	}
}

func TestCreateConversationShareRunsModerationBeforePersist(t *testing.T) {
	moderationServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/moderations" {
			t.Fatalf("unexpected moderation path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"flagged":true,"categories":{"self-harm":true},"category_scores":{"self-harm":0.91}}]}`))
	}))
	defer moderationServer.Close()

	repo := &shareModerationRepo{
		conversation: model.Conversation{ID: 11, UserID: 7, PublicID: "conv_1", Title: "Share me", Model: "model-a"},
		messages: []model.Message{
			{ID: 1, ConversationID: 11, UserID: 7, PublicID: "msg_user", Role: "user", Content: "unsafe content"},
			{ID: 2, ConversationID: 11, UserID: 7, PublicID: "msg_assistant", ParentPublicID: "msg_user", Role: "assistant", Content: "assistant reply"},
		},
	}
	service := &Service{
		cfg: config.NewRuntime(config.Config{
			ModerationEnabled:   true,
			ModerationBaseURL:   moderationServer.URL,
			ModerationModel:     "guard-model",
			ModerationThreshold: 0.5,
		}),
		repo: repo,
	}

	_, err := service.CreateConversationShare(context.Background(), 7, "conv_1", ConversationShareOptions{})

	if err != ErrModerationBlocked {
		t.Fatalf("CreateConversationShare() error = %v, want ErrModerationBlocked", err)
	}
	if repo.replaced {
		t.Fatal("share was persisted before moderation passed")
	}
	if repo.moderationEvent == nil {
		t.Fatal("expected moderation event to be recorded")
	}
	if repo.moderationEvent.Direction != "share" || !repo.moderationEvent.Flagged {
		t.Fatalf("moderation event = %#v", repo.moderationEvent)
	}
}

type shareModerationRepo struct {
	repository.ConversationRepository
	conversation    model.Conversation
	messages        []model.Message
	moderationEvent *model.ModerationEvent
	replaced        bool
}

func (r *shareModerationRepo) GetConversationByPublicID(_ context.Context, publicID string, userID uint) (*model.Conversation, error) {
	if r.conversation.PublicID != publicID || r.conversation.UserID != userID {
		return nil, repository.ErrNotFound
	}
	item := r.conversation
	return &item, nil
}

func (r *shareModerationRepo) ListMessagesForShare(_ context.Context, conversationID uint, publicIDs []string) ([]model.Message, error) {
	if conversationID != r.conversation.ID {
		return nil, repository.ErrNotFound
	}
	if len(publicIDs) == 0 {
		return append([]model.Message(nil), r.messages...), nil
	}
	wanted := make(map[string]struct{}, len(publicIDs))
	for _, id := range publicIDs {
		wanted[strings.TrimSpace(id)] = struct{}{}
	}
	result := make([]model.Message, 0, len(r.messages))
	for _, message := range r.messages {
		if _, ok := wanted[message.PublicID]; ok {
			result = append(result, message)
		}
	}
	return result, nil
}

func (r *shareModerationRepo) CreateModerationEvent(_ context.Context, event *model.ModerationEvent) error {
	copied := *event
	r.moderationEvent = &copied
	return nil
}

func (r *shareModerationRepo) ReplaceActiveConversationShare(_ context.Context, _ *model.ConversationShare) error {
	r.replaced = true
	return nil
}
