package conversation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	appnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/notification"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

const (
	moderationModeModerations    = "moderations"
	moderationModeChatClassifier = "chat_classifier"
	moderationFailOpen           = "fail_open"
	moderationFailClose          = "fail_close"
	moderationDispositionLimit   = "rate_limited"
	moderationDispositionSuspend = "suspended"
	moderationEventPolicyHit     = "policy_hit"
	moderationEventEngineError   = "engine_error"
)

const defaultModerationClassifierTemplate = `You are a content policy classifier. Review the {{DIRECTION}} content and return only JSON with this shape: {"flagged": boolean, "score": number, "categories": object, "reason": string}. Use score from 0 to 1.`

const (
	moderationSnapshotMaxRunes         = 2000
	defaultModerationOutputWindowChars = 800
	moderationVisibleSuspensionReason  = "多次发送不适宜内容"
	moderationAutoLimitDetail          = "系统已临时限制账号使用。"
	moderationAutoSuspendDetail        = "系统已暂停账号使用。"
	moderationAutoLimitRevokeReason    = "content_policy_auto_limit"
	moderationAutoSuspendRevokeReason  = "content_policy_auto_suspension"
	moderationNotificationSource       = "moderation"
	moderationNotificationLink         = "/notifications"
	moderationLimitNotificationTitle   = "账号已临时限速"
	moderationLimitNotificationBody    = "由于近期多次触发内容检查，账号请求频率已临时降低。"
	moderationSuspendNotificationTitle = "账号已暂停"
	moderationSuspendNotificationBody  = "由于近期多次触发内容检查，账号已暂停使用。"
	moderationReleaseNotificationTitle = "账号限制已解除"
	moderationReleaseNotificationBody  = "内容安全处置已解除，你可以继续正常使用账号。"
)

type moderationCheckResult struct {
	Flagged        bool
	Score          float64
	Threshold      float64
	Model          string
	CategoriesJSON string
	Reason         string
}

type moderationRuntimeConfig struct {
	Enabled                  bool
	BaseURL                  string
	APIKey                   string
	Model                    string
	Threshold                float64
	Action                   string
	TimeoutSeconds           int
	Mode                     string
	FailStrategy             string
	ClassifierTemplate       string
	AutoWindowHours          int
	AutoLimitThreshold       int
	AutoSuspendThreshold     int
	AutoLimitRPM             int
	AutoLimitDurationMinutes int
	OutputWindowChars        int
}

type moderationRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type moderationResponse struct {
	Results []struct {
		Flagged        bool               `json:"flagged"`
		Categories     map[string]bool    `json:"categories"`
		CategoryScores map[string]float64 `json:"category_scores"`
	} `json:"results"`
}

type chatClassifierRequest struct {
	Model          string                  `json:"model"`
	Messages       []chatClassifierMessage `json:"messages"`
	Temperature    float64                 `json:"temperature"`
	ResponseFormat map[string]string       `json:"response_format,omitempty"`
}

type chatClassifierMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatClassifierResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type chatClassifierDecoded struct {
	Flagged    bool                   `json:"flagged"`
	Score      float64                `json:"score"`
	Categories map[string]interface{} `json:"categories"`
	Reason     string                 `json:"reason"`
}

type moderationUserEnforcer interface {
	GetByID(ctx context.Context, userID uint) (*domainuser.User, error)
	UpdateUserStatus(ctx context.Context, userID uint, status string) error
	SetUserSuspension(ctx context.Context, userID uint, reason string, detail string, suspendedAt *time.Time, suspendedBy *uint) error
	RevokeAllSessions(ctx context.Context, userID uint, reason string) error
}

type moderationRateLimiter interface {
	SetUserRateLimitOverride(ctx context.Context, userID uint, rpm int, ttl time.Duration) error
	GetUserRateLimitOverride(ctx context.Context, userID uint) (int, bool, error)
	ClearUserRateLimitOverride(ctx context.Context, userID uint) error
}

func normalizeModerationRuntimeConfig(cfg config.Config) moderationRuntimeConfig {
	result := moderationRuntimeConfig{
		Enabled:                  cfg.ModerationEnabled,
		BaseURL:                  strings.TrimRight(strings.TrimSpace(cfg.ModerationBaseURL), "/"),
		APIKey:                   strings.TrimSpace(cfg.ModerationAPIKey),
		Model:                    strings.TrimSpace(cfg.ModerationModel),
		Threshold:                cfg.ModerationThreshold,
		Action:                   strings.TrimSpace(cfg.ModerationAction),
		TimeoutSeconds:           cfg.ModerationTimeoutSeconds,
		Mode:                     strings.TrimSpace(cfg.ModerationMode),
		FailStrategy:             strings.TrimSpace(cfg.ModerationFailStrategy),
		ClassifierTemplate:       strings.TrimSpace(cfg.ModerationClassifierTemplate),
		AutoWindowHours:          cfg.ModerationAutoWindowHours,
		AutoLimitThreshold:       cfg.ModerationAutoLimitThreshold,
		AutoSuspendThreshold:     cfg.ModerationAutoSuspendThreshold,
		AutoLimitRPM:             cfg.ModerationAutoLimitRPM,
		AutoLimitDurationMinutes: cfg.ModerationAutoLimitDurationMinutes,
		OutputWindowChars:        cfg.ModerationOutputWindowChars,
	}
	if result.Model == "" {
		result.Model = "omni-moderation-latest"
	}
	if result.Threshold <= 0 || result.Threshold > 1 {
		result.Threshold = 0.5
	}
	if result.Action == "" {
		result.Action = "block"
	}
	if result.TimeoutSeconds <= 0 {
		result.TimeoutSeconds = 10
	}
	switch result.Mode {
	case moderationModeModerations, moderationModeChatClassifier:
	default:
		result.Mode = moderationModeModerations
	}
	switch result.FailStrategy {
	case moderationFailOpen, moderationFailClose:
	default:
		result.FailStrategy = moderationFailOpen
	}
	if result.ClassifierTemplate == "" {
		result.ClassifierTemplate = defaultModerationClassifierTemplate
	}
	if result.AutoWindowHours <= 0 {
		result.AutoWindowHours = 24
	}
	if result.AutoLimitThreshold < 0 {
		result.AutoLimitThreshold = 0
	}
	if result.AutoSuspendThreshold < 0 {
		result.AutoSuspendThreshold = 0
	}
	if result.AutoLimitRPM <= 0 {
		result.AutoLimitRPM = 5
	}
	if result.AutoLimitDurationMinutes <= 0 {
		result.AutoLimitDurationMinutes = 60
	}
	if result.OutputWindowChars <= 0 {
		result.OutputWindowChars = defaultModerationOutputWindowChars
	}
	return result
}

func shouldBlockOnModerationError(strategy string) bool {
	return strings.TrimSpace(strategy) == moderationFailClose
}

func buildChatClassifierRequest(modelName string, template string, direction string, content string) ([]byte, error) {
	systemPrompt := strings.TrimSpace(template)
	if systemPrompt == "" {
		systemPrompt = defaultModerationClassifierTemplate
	}
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{DIRECTION}}", strings.TrimSpace(direction))
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{CONTENT}}", strings.TrimSpace(content))
	return json.Marshal(chatClassifierRequest{
		Model: strings.TrimSpace(modelName),
		Messages: []chatClassifierMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: strings.TrimSpace(content)},
		},
		Temperature:    0,
		ResponseFormat: map[string]string{"type": "json_object"},
	})
}

func parseChatClassifierContent(content string, threshold float64, modelName string) (*moderationCheckResult, error) {
	var decoded chatClassifierDecoded
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &decoded); err != nil {
		return nil, err
	}
	result := &moderationCheckResult{
		Flagged:        decoded.Flagged || decoded.Score >= threshold,
		Score:          decoded.Score,
		Threshold:      threshold,
		Model:          strings.TrimSpace(modelName),
		CategoriesJSON: "{}",
		Reason:         strings.TrimSpace(decoded.Reason),
	}
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 1 {
		result.Score = 1
	}
	if raw, err := json.Marshal(map[string]interface{}{
		"categories": decoded.Categories,
	}); err == nil {
		result.CategoriesJSON = string(raw)
	}
	return result, nil
}

func (s *Service) checkModeration(ctx context.Context, direction string, content string) (*moderationCheckResult, error) {
	cfg := normalizeModerationRuntimeConfig(s.cfg.Snapshot())
	if !cfg.Enabled {
		return nil, nil
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil
	}
	if cfg.BaseURL == "" {
		return nil, nil
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if cfg.Mode == moderationModeChatClassifier {
		return s.checkChatClassifierModeration(callCtx, cfg, direction, content)
	}
	return s.checkOpenAIModeration(callCtx, cfg, direction, content)
}

func (s *Service) checkOpenAIModeration(ctx context.Context, cfg moderationRuntimeConfig, direction string, content string) (*moderationCheckResult, error) {
	body, err := json.Marshal(moderationRequest{Model: cfg.Model, Input: content})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/moderations", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	snap := s.cfg.Snapshot()
	resp, err := security.NewOutboundHTTPClient(snap.Env, snap.SSRFProtectionEnabled, timeout).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("moderation request failed: %d", resp.StatusCode)
	}
	var decoded moderationResponse
	if err = json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	result := moderationCheckResult{Threshold: cfg.Threshold, Model: cfg.Model, CategoriesJSON: "{}", Reason: direction}
	if len(decoded.Results) == 0 {
		return &result, nil
	}
	first := decoded.Results[0]
	for _, score := range first.CategoryScores {
		if score > result.Score {
			result.Score = score
		}
	}
	result.Flagged = first.Flagged || result.Score >= cfg.Threshold
	if raw, err := json.Marshal(map[string]interface{}{
		"categories":      first.Categories,
		"category_scores": first.CategoryScores,
	}); err == nil {
		result.CategoriesJSON = string(raw)
	}
	return &result, nil
}

func (s *Service) checkChatClassifierModeration(ctx context.Context, cfg moderationRuntimeConfig, direction string, content string) (*moderationCheckResult, error) {
	body, err := buildChatClassifierRequest(cfg.Model, cfg.ClassifierTemplate, direction, content)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	snap := s.cfg.Snapshot()
	resp, err := security.NewOutboundHTTPClient(snap.Env, snap.SSRFProtectionEnabled, timeout).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("moderation classifier request failed: %d", resp.StatusCode)
	}
	var decoded chatClassifierResponse
	if err = json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if len(decoded.Choices) == 0 {
		return &moderationCheckResult{Threshold: cfg.Threshold, Model: cfg.Model, CategoriesJSON: "{}", Reason: direction}, nil
	}
	result, err := parseChatClassifierContent(decoded.Choices[0].Message.Content, cfg.Threshold, cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("parse moderation classifier content failed: %w", err)
	}
	if result.Reason == "" {
		result.Reason = direction
	}
	return result, nil
}

func buildModerationContentSnapshot(content string) (string, string, bool) {
	value := strings.TrimSpace(content)
	if value == "" {
		return "", "", false
	}
	sum := sha256.Sum256([]byte(value))
	runes := []rune(value)
	truncated := false
	if len(runes) > moderationSnapshotMaxRunes {
		value = string(runes[:moderationSnapshotMaxRunes])
		truncated = true
	}
	return value, hex.EncodeToString(sum[:]), truncated
}

func moderationEventType(reason string) string {
	if strings.TrimSpace(reason) == "check_failed" {
		return moderationEventEngineError
	}
	return moderationEventPolicyHit
}

func (s *Service) recordModerationEvent(ctx context.Context, input SendMessageInput, messageID uint, runID string, direction string, result *moderationCheckResult, reason string, content string) {
	if result == nil {
		return
	}
	snapshot, contentHash, snapshotTruncated := buildModerationContentSnapshot(content)
	event := &model.ModerationEvent{
		UserID:            input.UserID,
		ConversationID:    input.ConversationID,
		MessageID:         messageID,
		RunID:             strings.TrimSpace(runID),
		Direction:         direction,
		Action:            strings.TrimSpace(s.cfg.Snapshot().ModerationAction),
		Model:             result.Model,
		Score:             result.Score,
		Threshold:         result.Threshold,
		Flagged:           result.Flagged,
		CategoriesJSON:    result.CategoriesJSON,
		Reason:            strings.TrimSpace(reason),
		EventType:         moderationEventType(reason),
		ContentSnapshot:   snapshot,
		ContentHash:       contentHash,
		SnapshotTruncated: snapshotTruncated,
	}
	if event.Action == "" {
		event.Action = "block"
	}
	if event.Reason == "" {
		event.Reason = result.Reason
	}
	if err := s.repo.CreateModerationEvent(ctx, event); err == nil && event.Flagged && event.EventType == moderationEventPolicyHit {
		if disposition, appliedAt, dispositionErr := s.applyModerationAutoDisposition(ctx, input.UserID); dispositionErr == nil && disposition != "" {
			event.Disposition = disposition
			event.DispositionAppliedAt = appliedAt
			_, _ = s.repo.UpdateModerationEventDisposition(ctx, event.ID, disposition, appliedAt)
		}
	}
}

func (s *Service) moderationFailureResult(err error) *moderationCheckResult {
	cfg := normalizeModerationRuntimeConfig(s.cfg.Snapshot())
	return &moderationCheckResult{
		Flagged:        shouldBlockOnModerationError(cfg.FailStrategy),
		Threshold:      cfg.Threshold,
		Model:          cfg.Model,
		CategoriesJSON: "{}",
		Reason:         err.Error(),
	}
}

func (s *Service) shouldFailCloseModeration() bool {
	cfg := normalizeModerationRuntimeConfig(s.cfg.Snapshot())
	return shouldBlockOnModerationError(cfg.FailStrategy)
}

func (s *Service) applyModerationAutoDisposition(ctx context.Context, userID uint) (string, *time.Time, error) {
	if s.userEnforcer == nil || userID == 0 {
		return "", nil, nil
	}
	cfg := normalizeModerationRuntimeConfig(s.cfg.Snapshot())
	if cfg.AutoLimitThreshold <= 0 && cfg.AutoSuspendThreshold <= 0 {
		return "", nil, nil
	}
	user, err := s.userEnforcer.GetByID(ctx, userID)
	if err != nil || user == nil || domainuser.IsAdminRole(user.Role) {
		return "", nil, err
	}
	count, err := s.repo.CountFlaggedModerationEvents(ctx, userID, time.Now().Add(-time.Duration(cfg.AutoWindowHours)*time.Hour))
	if err != nil {
		return "", nil, err
	}
	now := time.Now().UTC()
	if cfg.AutoSuspendThreshold > 0 && count >= int64(cfg.AutoSuspendThreshold) &&
		(user.Status == domainuser.StatusActive || user.Status == domainuser.StatusLocked) {
		if err = s.userEnforcer.UpdateUserStatus(ctx, userID, domainuser.StatusSuspended); err != nil {
			return "", nil, err
		}
		if err = s.userEnforcer.SetUserSuspension(ctx, userID, moderationVisibleSuspensionReason, moderationAutoSuspendDetail, &now, nil); err != nil {
			return "", nil, err
		}
		if err = s.userEnforcer.RevokeAllSessions(ctx, userID, moderationAutoSuspendRevokeReason); err != nil {
			return "", nil, err
		}
		s.notifyModerationAutoDisposition(ctx, userID, moderationDispositionSuspend, now)
		return moderationDispositionSuspend, &now, nil
	}
	if cfg.AutoLimitThreshold > 0 && count >= int64(cfg.AutoLimitThreshold) && user.Status == domainuser.StatusActive && s.rateLimiter != nil {
		ttl := time.Duration(cfg.AutoLimitDurationMinutes) * time.Minute
		_, overrideExisted, _ := s.rateLimiter.GetUserRateLimitOverride(ctx, userID)
		if err = s.rateLimiter.SetUserRateLimitOverride(ctx, userID, cfg.AutoLimitRPM, ttl); err != nil {
			return "", nil, err
		}
		if !overrideExisted {
			s.notifyModerationAutoDisposition(ctx, userID, moderationDispositionLimit, now)
		}
		return moderationDispositionLimit, &now, nil
	}
	return "", nil, nil
}

func (s *Service) notifyModerationAutoDisposition(ctx context.Context, userID uint, disposition string, appliedAt time.Time) {
	if s.moderationNotifier == nil || userID == 0 {
		return
	}
	input, ok := moderationDispositionNotification(disposition, appliedAt)
	if !ok {
		return
	}
	_, _ = s.moderationNotifier.CreateSystemNotification(ctx, userID, input)
}

func moderationDispositionNotification(disposition string, appliedAt time.Time) (appnotification.SystemNotificationInput, bool) {
	input := appnotification.SystemNotificationInput{
		Type:      domainnotification.TypeModeration,
		ActionURL: moderationNotificationLink,
		Source:    moderationNotificationSource,
		SourceID:  fmt.Sprintf("%s:%d", strings.TrimSpace(disposition), appliedAt.Unix()),
		Metadata: map[string]any{
			"disposition": disposition,
			"appliedAt":   appliedAt.UTC().Format(time.RFC3339),
		},
	}
	switch disposition {
	case moderationDispositionLimit:
		input.Title = moderationLimitNotificationTitle
		input.Body = moderationLimitNotificationBody
	case moderationDispositionSuspend:
		input.Title = moderationSuspendNotificationTitle
		input.Body = moderationSuspendNotificationBody
	default:
		return appnotification.SystemNotificationInput{}, false
	}
	return input, true
}

func (s *Service) notifyModerationDispositionRelease(ctx context.Context, userID uint, eventID uint, disposition string, releasedAt time.Time) {
	if s.moderationNotifier == nil || userID == 0 {
		return
	}
	input, ok := moderationReleaseNotification(eventID, disposition, releasedAt)
	if !ok {
		return
	}
	_, _ = s.moderationNotifier.CreateSystemNotification(ctx, userID, input)
}

func moderationReleaseNotification(eventID uint, disposition string, releasedAt time.Time) (appnotification.SystemNotificationInput, bool) {
	if eventID == 0 {
		return appnotification.SystemNotificationInput{}, false
	}
	return appnotification.SystemNotificationInput{
		Type:      domainnotification.TypeModeration,
		Title:     moderationReleaseNotificationTitle,
		Body:      moderationReleaseNotificationBody,
		ActionURL: moderationNotificationLink,
		Source:    moderationNotificationSource,
		SourceID:  fmt.Sprintf("release:%d", eventID),
		Metadata: map[string]any{
			"disposition": strings.TrimSpace(disposition),
			"releasedAt":  releasedAt.UTC().Format(time.RFC3339),
		},
	}, true
}

func (s *Service) CheckContentForPolicy(ctx context.Context, userID uint, content string, direction string) error {
	return s.checkContentForPolicy(ctx, userID, 0, 0, "", content, direction)
}

func (s *Service) checkContentForPolicy(ctx context.Context, userID uint, conversationID uint, messageID uint, runID string, content string, direction string) error {
	result, err := s.checkModeration(ctx, direction, content)
	input := SendMessageInput{UserID: userID, ConversationID: conversationID}
	if err != nil {
		s.recordModerationEvent(ctx, input, messageID, runID, direction, s.moderationFailureResult(err), "check_failed", content)
		if s.shouldFailCloseModeration() {
			return ErrModerationBlocked
		}
		return nil
	}
	if result != nil && result.Flagged {
		s.recordModerationEvent(ctx, input, messageID, runID, direction, result, "blocked", content)
		return ErrModerationBlocked
	}
	return nil
}
