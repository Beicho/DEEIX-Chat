package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/pkg/conv"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/requestmeta"
	"github.com/google/uuid"
)

const (
	// maxInvitationBatchSize 单次批量生成上限。
	maxInvitationBatchSize = 1000
	// pendingRegistrationTTL 第三方登录待补邀请码的有效期。
	pendingRegistrationTTL = 15 * time.Minute
)

var (
	// ErrInvitationCodeRequired 需要邀请码才能完成注册。
	ErrInvitationCodeRequired = errors.New("invitation code is required")
	// ErrInvitationCodeInvalid 邀请码不存在、已停用、已过期或已用完。
	ErrInvitationCodeInvalid = errors.New("invitation code is invalid")
	// ErrPendingRegistrationInvalid 注册令牌无效或已过期。
	ErrPendingRegistrationInvalid = errors.New("pending registration is invalid")
)

// ProviderInvitationRequiredError 表示第三方登录首次注册需要补交邀请码。
type ProviderInvitationRequiredError struct {
	ProviderSlug string
}

func (e *ProviderInvitationRequiredError) Error() string {
	return "invitation code is required"
}

func (e *ProviderInvitationRequiredError) Unwrap() error {
	return ErrInvitationCodeRequired
}

// CreateInvitationCodes 批量生成邀请码，返回的明文只在本次响应中可见。
func (s *Service) CreateInvitationCodes(
	ctx context.Context,
	actorUserID uint,
	label string,
	count int,
	maxUses int,
	expiresAt *time.Time,
) ([]InvitationCodeResult, error) {
	if actorUserID == 0 {
		return nil, fmt.Errorf("admin permission required")
	}
	if count <= 0 {
		count = 1
	}
	if count > maxInvitationBatchSize {
		return nil, fmt.Errorf("invitation batch size exceeds %d", maxInvitationBatchSize)
	}
	if maxUses < 0 || maxUses > 100000 {
		return nil, fmt.Errorf("invalid invitation code max uses")
	}
	if maxUses == 0 {
		maxUses = 1
	}
	normalizedLabel := strings.TrimSpace(label)
	if len(normalizedLabel) > 80 {
		return nil, fmt.Errorf("invalid invitation code label")
	}

	secret := s.cfg.Snapshot().JWTSecret
	results := make([]InvitationCodeResult, 0, count)
	for i := 0; i < count; i++ {
		code, err := generateInvitationCode()
		if err != nil {
			return nil, err
		}
		item := &domainuser.InvitationCode{
			PublicID:  conv.NormalizePublicID(uuid.NewString()),
			CodeHash:  hashInvitationCode(secret, code),
			Label:     normalizedLabel,
			MaxUses:   maxUses,
			Enabled:   true,
			ExpiresAt: expiresAt,
			CreatedBy: actorUserID,
		}
		created, err := s.repo.CreateInvitationCode(ctx, item)
		if err != nil {
			return nil, err
		}
		result := toInvitationCodeResult(*created)
		result.Code = code
		results = append(results, result)
	}
	return results, nil
}

// invitationRequiredForProviderRegistration 判断第三方登录首次注册是否需要邀请码。
func (s *Service) invitationRequiredForProviderRegistration() bool {
	cfg := s.cfg.Snapshot()
	return cfg.InviteRegistrationRequired && cfg.InviteProviderRegistration
}

// invitationRequiredForEmailRegistration 判断邮箱注册是否需要邀请码。
func (s *Service) invitationRequiredForEmailRegistration() bool {
	cfg := s.cfg.Snapshot()
	return cfg.InviteRegistrationRequired && cfg.EmailRegistrationEnabled
}

// consumeInvitationCode 校验并占用一个邀请码名额。
func (s *Service) consumeInvitationCode(ctx context.Context, code string) error {
	normalized := normalizeInvitationCode(code)
	if normalized == "" {
		return ErrInvitationCodeRequired
	}
	hash := hashInvitationCode(s.cfg.Snapshot().JWTSecret, normalized)
	if _, err := s.repo.ConsumeInvitationCode(ctx, hash, time.Now()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvitationCodeInvalid
		}
		return err
	}
	return nil
}

// pendingProviderRegistration 承载第三方授权已完成、等待补交邀请码的注册上下文。
type pendingProviderRegistration struct {
	ProviderSlug  string `json:"provider_slug"`
	Subject       string `json:"subject"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name"`
	AvatarURL     string `json:"avatar_url"`
	EmailVerified bool   `json:"email_verified"`
	ProfileJSON   string `json:"profile_json"`
	ExpiresAt     int64  `json:"expires_at"`
}

func (s *Service) signPendingRegistration(payload pendingProviderRegistration) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(encoded)
	return body + "." + pendingRegistrationSignature(s.cfg.Snapshot().JWTSecret, body), nil
}

func (s *Service) verifyPendingRegistration(token string) (*pendingProviderRegistration, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, ErrPendingRegistrationInvalid
	}
	expected := pendingRegistrationSignature(s.cfg.Snapshot().JWTSecret, parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, ErrPendingRegistrationInvalid
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrPendingRegistrationInvalid
	}
	var payload pendingProviderRegistration
	if err = json.Unmarshal(decoded, &payload); err != nil {
		return nil, ErrPendingRegistrationInvalid
	}
	if payload.ExpiresAt <= 0 || time.Now().After(time.Unix(payload.ExpiresAt, 0)) {
		return nil, ErrPendingRegistrationInvalid
	}
	if strings.TrimSpace(payload.ProviderSlug) == "" || strings.TrimSpace(payload.Subject) == "" {
		return nil, ErrPendingRegistrationInvalid
	}
	return &payload, nil
}

func pendingRegistrationSignature(secret string, body string) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)+":pending-registration"))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

// CompletePendingProviderRegistration 使用待注册令牌与邀请码完成第三方注册。
func (s *Service) CompletePendingProviderRegistration(
	ctx context.Context,
	token string,
	invitationCode string,
	requestID string,
	auditCtx requestmeta.SessionAuditContext,
) (*LoginResult, error) {
	if !s.cfg.Snapshot().ThirdPartyLoginEnabled {
		return nil, fmt.Errorf("third-party login is disabled")
	}
	payload, err := s.verifyPendingRegistration(token)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(invitationCode) == "" {
		return nil, ErrInvitationCodeRequired
	}
	provider, err := s.repo.GetIdentityProviderBySlug(ctx, payload.ProviderSlug)
	if err != nil {
		return nil, err
	}
	if !provider.LoginEnabled || !provider.RegistrationEnabled {
		return nil, fmt.Errorf("provider registration is disabled")
	}

	userItem, err := s.resolveProviderUser(
		ctx,
		*provider,
		payload.Subject,
		payload.Email,
		payload.DisplayName,
		payload.AvatarURL,
		payload.EmailVerified,
		payload.ProfileJSON,
		providerIntentRegister,
		invitationCode,
	)
	if err != nil {
		return nil, err
	}

	normalizedAuditCtx := s.resolveSessionAuditContext(ctx, auditCtx)
	result, err := s.issueLoginResult(ctx, userItem, normalizedAuditCtx, time.Now())
	if err != nil {
		return nil, err
	}
	s.RecordAuthEvent(
		ctx,
		result.User.ID,
		requestID,
		"provider_register",
		"success",
		"",
		normalizedAuditCtx.ClientIP,
		normalizedAuditCtx.UserAgent,
		marshalAuthEventDetail(map[string]interface{}{
			"provider":   provider.Slug,
			"subject":    payload.Subject,
			"session_id": result.SessionID,
			"invited":    true,
		}),
	)
	return result, nil
}
