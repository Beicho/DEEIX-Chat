package alerting

import (
	"context"
	"strings"

	domainsettings "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/settings"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/pkg/secretbox"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

// settingsNamespace 是告警配置在 system_settings 表中的 namespace。
const settingsNamespace = "alerting"

// sensitiveKeys 标记需加密存储的告警配置键。
var sensitiveKeys = map[string]struct{}{
	KeyTelegramBotToken: {},
}

// settingsValueType 各配置键的值类型（用于落库展示）。
var settingsValueType = map[string]string{
	KeyEnabled:          "bool",
	KeyEnabledNotifiers: "json",
	KeyTelegramBotToken: "string",
	KeyTelegramChatID:   "string",
	KeyWebhookURL:       "string",
	KeyDebounceSeconds:  "int",
}

// SettingsConfigStore 基于 system_settings 表实现告警配置读写，敏感项 AES-GCM 加密。
type SettingsConfigStore struct {
	repo              repository.SettingsRepository
	dataEncryptionKey string
}

// NewSettingsConfigStore 创建配置存储。
func NewSettingsConfigStore(repo repository.SettingsRepository, dataEncryptionKey string) *SettingsConfigStore {
	return &SettingsConfigStore{repo: repo, dataEncryptionKey: strings.TrimSpace(dataEncryptionKey)}
}

func isSensitiveKey(key string) bool {
	_, ok := sensitiveKeys[key]
	return ok
}

// Load 读取 alerting namespace 全部配置，敏感项解密。
func (s *SettingsConfigStore) Load(ctx context.Context) (map[string]string, error) {
	items, err := s.repo.ListByNamespace(ctx, settingsNamespace)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(items))
	for _, item := range items {
		value := item.Value
		if isSensitiveKey(item.Key) && strings.TrimSpace(value) != "" {
			decrypted, decErr := secretbox.DecryptString(s.dataEncryptionKey, value)
			if decErr != nil {
				// 解密失败视为未配置，避免把密文当 token 发出去。
				continue
			}
			value = decrypted
		}
		result[item.Key] = strings.TrimSpace(value)
	}
	return result, nil
}

// Save 写入配置补丁，敏感项加密；Clear 表示删除该键。
func (s *SettingsConfigStore) Save(ctx context.Context, patches []SettingPatch) error {
	upserts := make([]domainsettings.SystemSetting, 0, len(patches))
	for _, patch := range patches {
		key := strings.TrimSpace(patch.Key)
		if key == "" {
			continue
		}
		if patch.Clear {
			if err := s.repo.Delete(ctx, settingsNamespace, key); err != nil {
				return err
			}
			continue
		}
		value := patch.Value
		if isSensitiveKey(key) && strings.TrimSpace(value) != "" {
			encrypted, err := secretbox.EncryptString(s.dataEncryptionKey, value)
			if err != nil {
				return err
			}
			value = encrypted
		}
		upserts = append(upserts, domainsettings.SystemSetting{
			Namespace: settingsNamespace,
			Key:       key,
			Value:     value,
			ValueType: valueTypeFor(key),
		})
	}
	if len(upserts) == 0 {
		return nil
	}
	return s.repo.Upsert(ctx, upserts)
}

func valueTypeFor(key string) string {
	if vt, ok := settingsValueType[key]; ok {
		return vt
	}
	return "string"
}

var _ ConfigStore = (*SettingsConfigStore)(nil)
