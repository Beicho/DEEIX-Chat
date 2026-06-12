package conversation

import (
	"encoding/json"
	"testing"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
)

func TestNormalizeModerationRuntimeConfigDefaults(t *testing.T) {
	cfg := normalizeModerationRuntimeConfig(config.Config{})

	if cfg.Mode != moderationModeModerations {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, moderationModeModerations)
	}
	if cfg.FailStrategy != moderationFailOpen {
		t.Fatalf("FailStrategy = %q, want %q", cfg.FailStrategy, moderationFailOpen)
	}
	if cfg.Model != "omni-moderation-latest" {
		t.Fatalf("Model = %q", cfg.Model)
	}
	if cfg.Threshold != 0.5 {
		t.Fatalf("Threshold = %v, want 0.5", cfg.Threshold)
	}
	if cfg.TimeoutSeconds != 10 {
		t.Fatalf("TimeoutSeconds = %d, want 10", cfg.TimeoutSeconds)
	}
	if cfg.AutoWindowHours != 24 {
		t.Fatalf("AutoWindowHours = %d, want 24", cfg.AutoWindowHours)
	}
	if cfg.ClassifierTemplate == "" {
		t.Fatal("ClassifierTemplate is empty")
	}
}

func TestShouldBlockOnModerationError(t *testing.T) {
	if shouldBlockOnModerationError(moderationFailOpen) {
		t.Fatal("fail_open should not block on moderation errors")
	}
	if !shouldBlockOnModerationError(moderationFailClose) {
		t.Fatal("fail_close should block on moderation errors")
	}
	if shouldBlockOnModerationError("unexpected") {
		t.Fatal("unexpected strategy should normalize to fail_open")
	}
}

func TestParseChatClassifierContent(t *testing.T) {
	result, err := parseChatClassifierContent(`{"flagged":true,"score":0.84,"categories":{"hate":true},"reason":"policy"}`, 0.7, "classifier")
	if err != nil {
		t.Fatalf("parseChatClassifierContent returned error: %v", err)
	}
	if !result.Flagged {
		t.Fatal("Flagged = false, want true")
	}
	if result.Score != 0.84 {
		t.Fatalf("Score = %v, want 0.84", result.Score)
	}
	if result.Reason != "policy" {
		t.Fatalf("Reason = %q, want policy", result.Reason)
	}
	var categories map[string]interface{}
	if err := json.Unmarshal([]byte(result.CategoriesJSON), &categories); err != nil {
		t.Fatalf("CategoriesJSON is invalid JSON: %v", err)
	}
	if categories["categories"] == nil {
		t.Fatalf("CategoriesJSON missing categories: %s", result.CategoriesJSON)
	}
}

func TestBuildChatClassifierRequestAppliesTemplateVariables(t *testing.T) {
	body, err := buildChatClassifierRequest("guard-model", "Review {{DIRECTION}}", "output", "bad content")
	if err != nil {
		t.Fatalf("buildChatClassifierRequest returned error: %v", err)
	}
	var decoded struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		ResponseFormat struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("request body is invalid JSON: %v", err)
	}
	if decoded.Model != "guard-model" {
		t.Fatalf("Model = %q", decoded.Model)
	}
	if len(decoded.Messages) != 2 {
		t.Fatalf("messages length = %d, want 2", len(decoded.Messages))
	}
	if decoded.Messages[0].Content != "Review output" {
		t.Fatalf("system content = %q", decoded.Messages[0].Content)
	}
	if decoded.Messages[1].Content != "bad content" {
		t.Fatalf("user content = %q", decoded.Messages[1].Content)
	}
	if decoded.ResponseFormat.Type != "json_object" {
		t.Fatalf("response_format.type = %q", decoded.ResponseFormat.Type)
	}
}
