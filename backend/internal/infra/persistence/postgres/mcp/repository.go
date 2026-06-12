package mcp

import (
	"context"
	"encoding/json"
	"time"

	domainmcp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/mcp"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateServer(ctx context.Context, input repository.CreateMCPServerInput) (*domainmcp.Server, error) {
	item := model.MCPServer{
		OwnerUserID:          input.OwnerUserID,
		Name:                 input.Name,
		BaseURL:              input.BaseURL,
		AuthTokenEnc:         input.AuthTokenEnc,
		HeadersJSON:          input.HeadersJSON,
		Status:               input.Status,
		TimeoutSeconds:       input.TimeoutSeconds,
		OAuthClientID:        input.OAuthClientID,
		OAuthClientSecretEnc: input.OAuthClientSecretEnc,
		OAuthAuthURL:         input.OAuthAuthURL,
		OAuthTokenURL:        input.OAuthTokenURL,
		OAuthScopes:          input.OAuthScopes,
		OAuthAccessTokenEnc:  input.OAuthAccessTokenEnc,
		OAuthRefreshTokenEnc: input.OAuthRefreshTokenEnc,
	}
	if err := r.db.WithContext(ctx).Create(&item).Error; err != nil {
		return nil, err
	}
	result := toDomainServer(item)
	return &result, nil
}

func (r *Repo) UpdateServer(ctx context.Context, serverID uint, input repository.UpdateMCPServerInput) (*domainmcp.Server, error) {
	updates := map[string]interface{}{}
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.BaseURL != nil {
		updates["base_url"] = *input.BaseURL
	}
	if input.AuthTokenEnc != nil {
		updates["auth_token_enc"] = *input.AuthTokenEnc
	}
	if input.HeadersJSON != nil {
		updates["headers_json"] = *input.HeadersJSON
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.TimeoutSeconds != nil {
		updates["timeout_seconds"] = *input.TimeoutSeconds
	}
	if input.OAuthClientID != nil {
		updates["oauth_client_id"] = *input.OAuthClientID
	}
	if input.OAuthClientSecretEnc != nil {
		updates["oauth_client_secret_enc"] = *input.OAuthClientSecretEnc
	}
	if input.OAuthAuthURL != nil {
		updates["oauth_auth_url"] = *input.OAuthAuthURL
	}
	if input.OAuthTokenURL != nil {
		updates["oauth_token_url"] = *input.OAuthTokenURL
	}
	if input.OAuthScopes != nil {
		updates["oauth_scopes"] = *input.OAuthScopes
	}
	if input.OAuthAccessTokenEnc != nil {
		updates["oauth_access_token_enc"] = *input.OAuthAccessTokenEnc
	}
	if input.OAuthRefreshTokenEnc != nil {
		updates["oauth_refresh_token_enc"] = *input.OAuthRefreshTokenEnc
	}
	if input.OAuthStatus != nil {
		updates["oauth_status"] = *input.OAuthStatus
	}
	if input.LastError != nil {
		updates["last_error"] = *input.LastError
	}
	if len(updates) > 0 {
		if err := r.db.WithContext(ctx).Model(&model.MCPServer{}).Where("id = ?", serverID).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return r.GetServer(ctx, serverID)
}

func (r *Repo) ListServers(ctx context.Context) ([]domainmcp.Server, error) {
	var rows []model.MCPServer
	if err := r.db.WithContext(ctx).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.hydrateServerActiveCounts(ctx, rows)
}

func (r *Repo) ListServersForUser(ctx context.Context, userID uint, includePlatform bool) ([]domainmcp.Server, error) {
	var rows []model.MCPServer
	query := r.db.WithContext(ctx).Order("owner_user_id asc, id asc")
	if includePlatform {
		query = query.Where("owner_user_id = ? OR owner_user_id = 0", userID)
	} else {
		query = query.Where("owner_user_id = ?", userID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.hydrateServerActiveCounts(ctx, rows)
}

func (r *Repo) hydrateServerActiveCounts(ctx context.Context, rows []model.MCPServer) ([]domainmcp.Server, error) {
	activeCounts := map[uint]int{}
	if len(rows) > 0 {
		serverIDs := make([]uint, 0, len(rows))
		for _, row := range rows {
			serverIDs = append(serverIDs, row.ID)
		}
		var counts []struct {
			ServerID uint
			Count    int
		}
		if err := r.db.WithContext(ctx).
			Model(&model.MCPTool{}).
			Select("server_id, count(*) as count").
			Where("server_id IN ? AND status = ?", serverIDs, "active").
			Group("server_id").
			Scan(&counts).Error; err != nil {
			return nil, err
		}
		for _, item := range counts {
			activeCounts[item.ServerID] = item.Count
		}
	}
	items := make([]domainmcp.Server, 0, len(rows))
	for _, row := range rows {
		item := toDomainServer(row)
		item.ActiveToolCount = activeCounts[row.ID]
		items = append(items, item)
	}
	return items, nil
}

func (r *Repo) GetServer(ctx context.Context, serverID uint) (*domainmcp.Server, error) {
	var row model.MCPServer
	if err := r.db.WithContext(ctx).First(&row, "id = ?", serverID).Error; err != nil {
		return nil, err
	}
	item := toDomainServer(row)
	return &item, nil
}

func (r *Repo) GetServerForUser(ctx context.Context, serverID uint, userID uint, includePlatform bool) (*domainmcp.Server, error) {
	var row model.MCPServer
	query := r.db.WithContext(ctx).Where("id = ?", serverID)
	if includePlatform {
		query = query.Where("owner_user_id = ? OR owner_user_id = 0", userID)
	} else {
		query = query.Where("owner_user_id = ?", userID)
	}
	if err := query.First(&row).Error; err != nil {
		return nil, err
	}
	item := toDomainServer(row)
	return &item, nil
}

func (r *Repo) DeleteServer(ctx context.Context, serverID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("server_id = ?", serverID).Delete(&model.MCPTool{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.MCPServer{}, "id = ?", serverID).Error
	})
}

func (r *Repo) ReplaceServerTools(ctx context.Context, serverID uint, tools []domainmcp.Tool) error {
	now := time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows := make([]model.MCPTool, 0, len(tools))
		names := make([]string, 0, len(tools))
		for _, tool := range tools {
			names = append(names, tool.Name)
			rows = append(rows, model.MCPTool{
				ServerID:        serverID,
				Name:            tool.Name,
				DisplayName:     tool.DisplayName,
				Description:     tool.Description,
				InputSchemaJSON: tool.InputSchemaJSON,
				Status:          tool.Status,
				DefaultEnabled:  tool.DefaultEnabled,
				RequiresConfirm: tool.RequiresConfirm,
				ToolKind:        defaultToolKind(tool.ToolKind),
			})
		}
		if len(rows) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "server_id"}, {Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"input_schema_json",
					"updated_at",
				}),
			}).Create(&rows).Error; err != nil {
				return err
			}
		}
		deleteQuery := tx.Where("server_id = ?", serverID)
		if len(names) > 0 {
			deleteQuery = deleteQuery.Where("name NOT IN ?", names)
		}
		if err := deleteQuery.Delete(&model.MCPTool{}).Error; err != nil {
			return err
		}
		return tx.Model(&model.MCPServer{}).Where("id = ?", serverID).Updates(map[string]interface{}{
			"tool_count":     len(tools),
			"last_synced_at": &now,
			"last_error":     "",
		}).Error
	})
}

func (r *Repo) ListTools(ctx context.Context, serverID uint, onlyActive bool) ([]domainmcp.Tool, error) {
	query := r.db.WithContext(ctx).Where("server_id = ?", serverID).Order("name asc")
	if onlyActive {
		query = query.Where("status = ?", "active")
	}
	var rows []model.MCPTool
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domainmcp.Tool, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDomainTool(row))
	}
	return items, nil
}

func (r *Repo) ListToolsByIDs(ctx context.Context, toolIDs []uint) ([]domainmcp.Tool, error) {
	if len(toolIDs) == 0 {
		return []domainmcp.Tool{}, nil
	}
	var rows []model.MCPTool
	if err := r.db.WithContext(ctx).Where("id IN ?", toolIDs).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domainmcp.Tool, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDomainTool(row))
	}
	return items, nil
}

func (r *Repo) ListToolsByIDsForUser(ctx context.Context, toolIDs []uint, userID uint) ([]domainmcp.Tool, error) {
	if len(toolIDs) == 0 {
		return []domainmcp.Tool{}, nil
	}
	var rows []model.MCPTool
	if err := r.db.WithContext(ctx).
		Model(&model.MCPTool{}).
		Select("mcp_tools.*").
		Joins("JOIN mcp_servers ON mcp_servers.id = mcp_tools.server_id").
		Where("mcp_tools.id IN ? AND (mcp_servers.owner_user_id = ? OR mcp_servers.owner_user_id = 0)", toolIDs, userID).
		Order("mcp_tools.id asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domainmcp.Tool, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDomainTool(row))
	}
	return items, nil
}

func (r *Repo) UpdateTool(ctx context.Context, toolID uint, input repository.UpdateMCPToolInput) (*domainmcp.Tool, error) {
	updates := map[string]interface{}{}
	if input.DisplayName != nil {
		updates["display_name"] = *input.DisplayName
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.DefaultEnabled != nil {
		updates["default_enabled"] = *input.DefaultEnabled
	}
	if input.RequiresConfirm != nil {
		updates["requires_confirm"] = *input.RequiresConfirm
	}
	if len(updates) > 0 {
		if err := r.db.WithContext(ctx).Model(&model.MCPTool{}).Where("id = ?", toolID).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	var row model.MCPTool
	if err := r.db.WithContext(ctx).First(&row, "id = ?", toolID).Error; err != nil {
		return nil, err
	}
	item := toDomainTool(row)
	return &item, nil
}

func (r *Repo) UpdateServerToolsStatus(ctx context.Context, serverID uint, toolIDs []uint, status string) ([]domainmcp.Tool, error) {
	if err := r.db.WithContext(ctx).
		Model(&model.MCPTool{}).
		Where("server_id = ? AND id IN ?", serverID, toolIDs).
		Update("status", status).Error; err != nil {
		return nil, err
	}
	return r.ListTools(ctx, serverID, false)
}

func (r *Repo) GetToolPreference(ctx context.Context, userID uint, conversationPublicID string) (*domainmcp.ToolPreference, error) {
	var row model.MCPToolPreference
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND conversation_public_id = ?", userID, conversationPublicID).
		First(&row).Error; err != nil {
		return nil, err
	}
	item := toDomainToolPreference(row)
	return &item, nil
}

func (r *Repo) UpsertToolPreference(ctx context.Context, input repository.UpsertMCPToolPreferenceInput) (*domainmcp.ToolPreference, error) {
	selectedJSON := encodeUintList(input.SelectedToolIDs)
	confirmedJSON := encodeUintList(input.ConfirmedToolIDs)
	row := model.MCPToolPreference{
		UserID:               input.UserID,
		ConversationPublicID: input.ConversationPublicID,
		SelectedToolIDsJSON:  selectedJSON,
		ConfirmedToolIDsJSON: confirmedJSON,
		WebSearchEnabled:     input.WebSearchEnabled,
		CodeSandboxEnabled:   input.CodeSandboxEnabled,
		ResearchMaxLLMCalls:  input.ResearchMaxLLMCalls,
		ResearchMaxToolCalls: input.ResearchMaxToolCalls,
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "conversation_public_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"selected_tool_ids_json",
			"confirmed_tool_ids_json",
			"web_search_enabled",
			"code_sandbox_enabled",
			"research_max_llm_calls",
			"research_max_tool_calls",
			"updated_at",
		}),
	}).Create(&row).Error; err != nil {
		return nil, err
	}
	return r.GetToolPreference(ctx, input.UserID, input.ConversationPublicID)
}

func toDomainServer(row model.MCPServer) domainmcp.Server {
	return domainmcp.Server{
		ID:                   row.ID,
		OwnerUserID:          row.OwnerUserID,
		Name:                 row.Name,
		BaseURL:              row.BaseURL,
		AuthTokenEnc:         row.AuthTokenEnc,
		HeadersJSON:          row.HeadersJSON,
		Status:               row.Status,
		TimeoutSeconds:       row.TimeoutSeconds,
		OAuthClientID:        row.OAuthClientID,
		OAuthClientSecretEnc: row.OAuthClientSecretEnc,
		OAuthAuthURL:         row.OAuthAuthURL,
		OAuthTokenURL:        row.OAuthTokenURL,
		OAuthScopes:          row.OAuthScopes,
		OAuthAccessTokenEnc:  row.OAuthAccessTokenEnc,
		OAuthRefreshTokenEnc: row.OAuthRefreshTokenEnc,
		OAuthTokenExpiresAt:  row.OAuthTokenExpiresAt,
		OAuthStatus:          row.OAuthStatus,
		ToolCount:            row.ToolCount,
		ActiveToolCount:      0,
		LastSyncedAt:         row.LastSyncedAt,
		LastError:            row.LastError,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func toDomainTool(row model.MCPTool) domainmcp.Tool {
	return domainmcp.Tool{
		ID:              row.ID,
		ServerID:        row.ServerID,
		Name:            row.Name,
		DisplayName:     row.DisplayName,
		Description:     row.Description,
		InputSchemaJSON: row.InputSchemaJSON,
		Status:          row.Status,
		DefaultEnabled:  row.DefaultEnabled,
		RequiresConfirm: row.RequiresConfirm,
		ToolKind:        defaultToolKind(row.ToolKind),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func toDomainToolPreference(row model.MCPToolPreference) domainmcp.ToolPreference {
	return domainmcp.ToolPreference{
		ID:                   row.ID,
		UserID:               row.UserID,
		ConversationPublicID: row.ConversationPublicID,
		SelectedToolIDs:      decodeUintList(row.SelectedToolIDsJSON),
		ConfirmedToolIDs:     decodeUintList(row.ConfirmedToolIDsJSON),
		WebSearchEnabled:     row.WebSearchEnabled,
		CodeSandboxEnabled:   row.CodeSandboxEnabled,
		ResearchMaxLLMCalls:  row.ResearchMaxLLMCalls,
		ResearchMaxToolCalls: row.ResearchMaxToolCalls,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func encodeUintList(items []uint) string {
	if items == nil {
		items = []uint{}
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func decodeUintList(raw string) []uint {
	var items []uint
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []uint{}
	}
	result := make([]uint, 0, len(items))
	seen := map[uint]struct{}{}
	for _, item := range items {
		if item == 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func defaultToolKind(value string) string {
	if value == "" {
		return "remote"
	}
	return value
}
