package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	domainmcp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/mcp"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
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
	var result domainmcp.Server
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxSortOrder int
		if err := tx.Model(&model.MCPServer{}).
			Select("COALESCE(MAX(sort_order), 0)").
			Scan(&maxSortOrder).Error; err != nil {
			return err
		}
		item := model.MCPServer{
			OwnerUserID:          input.OwnerUserID,
			Name:                 input.Name,
			BaseURL:              input.BaseURL,
			AuthTokenEnc:         input.AuthTokenEnc,
			HeadersJSON:          input.HeadersJSON,
			Status:               input.Status,
			SortOrder:            maxSortOrder + 100,
			TimeoutSeconds:       input.TimeoutSeconds,
			OAuthClientID:        input.OAuthClientID,
			OAuthClientSecretEnc: input.OAuthClientSecretEnc,
			OAuthAuthURL:         input.OAuthAuthURL,
			OAuthTokenURL:        input.OAuthTokenURL,
			OAuthScopes:          input.OAuthScopes,
			OAuthAccessTokenEnc:  input.OAuthAccessTokenEnc,
			OAuthRefreshTokenEnc: input.OAuthRefreshTokenEnc,
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		result = toDomainServer(item)
		return nil
	}); err != nil {
		return nil, err
	}
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
	return r.listServers(ctx, r.db)
}

func (r *Repo) listServers(ctx context.Context, db *gorm.DB) ([]domainmcp.Server, error) {
	var rows []model.MCPServer
	if err := db.WithContext(ctx).Order("sort_order asc").Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.hydrateServerActiveCounts(ctx, db, rows)
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
	return r.hydrateServerActiveCounts(ctx, r.db, rows)
}

func (r *Repo) hydrateServerActiveCounts(ctx context.Context, db *gorm.DB, rows []model.MCPServer) ([]domainmcp.Server, error) {
	activeCounts := map[uint]int{}
	metadataConfirmationServers := map[uint]bool{}
	if len(rows) > 0 {
		serverIDs := make([]uint, 0, len(rows))
		for _, row := range rows {
			serverIDs = append(serverIDs, row.ID)
		}
		var counts []struct {
			ServerID uint
			Count    int
		}
		if err := db.WithContext(ctx).
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
		var confirmationServerIDs []uint
		if err := db.WithContext(ctx).
			Model(&model.MCPTool{}).
			Distinct("server_id").
			Where("server_id IN ? AND (metadata_customized = ? OR metadata_customized IS NULL)", serverIDs, true).
			Pluck("server_id", &confirmationServerIDs).Error; err != nil {
			return nil, err
		}
		for _, serverID := range confirmationServerIDs {
			metadataConfirmationServers[serverID] = true
		}
	}
	items := make([]domainmcp.Server, 0, len(rows))
	for _, row := range rows {
		item := toDomainServer(row)
		item.ActiveToolCount = activeCounts[row.ID]
		item.RequiresToolMetadataSyncConfirmation = metadataConfirmationServers[row.ID]
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
	var activeToolCount int64
	if err := r.db.WithContext(ctx).
		Model(&model.MCPTool{}).
		Where("server_id = ? AND status = ?", serverID, "active").
		Count(&activeToolCount).Error; err != nil {
		return nil, err
	}
	var metadataConfirmationCount int64
	if err := r.db.WithContext(ctx).
		Model(&model.MCPTool{}).
		Where("server_id = ? AND (metadata_customized = ? OR metadata_customized IS NULL)", serverID, true).
		Count(&metadataConfirmationCount).Error; err != nil {
		return nil, err
	}
	item.ActiveToolCount = int(activeToolCount)
	item.RequiresToolMetadataSyncConfirmation = metadataConfirmationCount > 0
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
		toolIDs := make([]uint, 0)
		if err := tx.Model(&model.MCPTool{}).Where("server_id = ?", serverID).Pluck("id", &toolIDs).Error; err != nil {
			return err
		}
		if err := deleteConversationProjectMCPToolAssociations(tx, toolIDs); err != nil {
			return err
		}
		if err := tx.Where("server_id = ?", serverID).Delete(&model.MCPTool{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.MCPServer{}, "id = ?", serverID).Error
	})
}

func (r *Repo) ReplaceServerTools(ctx context.Context, serverID uint, tools []domainmcp.Tool, overwriteCustomizedMetadata bool) error {
	now := time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxSortOrder int
		if err := tx.Model(&model.MCPTool{}).
			Where("server_id = ?", serverID).
			Select("COALESCE(MAX(sort_order), 0)").
			Scan(&maxSortOrder).Error; err != nil {
			return err
		}
		rows := make([]model.MCPTool, 0, len(tools))
		names := make([]string, 0, len(tools))
		for index, tool := range tools {
			metadataCustomized := false
			attachmentInputMode := strings.TrimSpace(tool.AttachmentInputMode)
			if attachmentInputMode == "" {
				attachmentInputMode = domainmcp.AttachmentInputModeNone
			}
			names = append(names, tool.Name)
			rows = append(rows, model.MCPTool{
				ServerID:                 serverID,
				Name:                     tool.Name,
				DisplayName:              tool.DisplayName,
				Description:              tool.Description,
				MetadataCustomized:       &metadataCustomized,
				InputSchemaJSON:          tool.InputSchemaJSON,
				AttachmentInputMode:      attachmentInputMode,
				AttachmentArgument:       strings.TrimSpace(tool.AttachmentArgument),
				AttachmentEncoding:       strings.TrimSpace(tool.AttachmentEncoding),
				AttachmentPromptArgument: strings.TrimSpace(tool.AttachmentPromptArgument),
				Status:                   tool.Status,
				SortOrder:                maxSortOrder + (index+1)*100,
				DefaultEnabled:           tool.DefaultEnabled,
				RequiresConfirm:          tool.RequiresConfirm,
				ToolKind:                 defaultToolKind(tool.ToolKind),
			})
		}
		if len(rows) > 0 {
			targetColumn := func(name string) string {
				if tx.Dialector.Name() == "postgres" {
					return `"mcp_tools"."` + name + `"`
				}
				return `"` + name + `"`
			}
			metadataCustomizedColumn := targetColumn("metadata_customized")
			displayNameColumn := targetColumn("display_name")
			descriptionColumn := targetColumn("description")
			legacyMetadataDiffers := "(" + displayNameColumn + ` <> excluded."display_name" OR ` + descriptionColumn + ` <> excluded."description")`
			metadataAssignments := map[string]interface{}{
				"display_name":        gorm.Expr("CASE WHEN COALESCE(" + metadataCustomizedColumn + ", TRUE) THEN " + displayNameColumn + ` ELSE excluded."display_name" END`),
				"description":         gorm.Expr("CASE WHEN COALESCE(" + metadataCustomizedColumn + ", TRUE) THEN " + descriptionColumn + ` ELSE excluded."description" END`),
				"metadata_customized": gorm.Expr("CASE WHEN " + metadataCustomizedColumn + " IS NULL THEN " + legacyMetadataDiffers + " ELSE " + metadataCustomizedColumn + " END"),
			}
			if overwriteCustomizedMetadata {
				metadataAssignments = map[string]interface{}{
					"display_name":        gorm.Expr(`excluded."display_name"`),
					"description":         gorm.Expr(`excluded."description"`),
					"metadata_customized": false,
				}
			}
			metadataAssignments["input_schema_json"] = gorm.Expr(`excluded."input_schema_json"`)
			metadataAssignments["attachment_input_mode"] = gorm.Expr(`excluded."attachment_input_mode"`)
			metadataAssignments["attachment_argument"] = gorm.Expr(`excluded."attachment_argument"`)
			metadataAssignments["attachment_encoding"] = gorm.Expr(`excluded."attachment_encoding"`)
			metadataAssignments["attachment_prompt_argument"] = gorm.Expr(`excluded."attachment_prompt_argument"`)
			metadataAssignments["updated_at"] = gorm.Expr(`excluded."updated_at"`)
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "server_id"}, {Name: "name"}},
				DoUpdates: clause.Assignments(metadataAssignments),
			}).Create(&rows).Error; err != nil {
				return err
			}
		}
		staleToolIDs := make([]uint, 0)
		staleToolQuery := tx.Model(&model.MCPTool{}).Where("server_id = ?", serverID)
		if len(names) > 0 {
			staleToolQuery = staleToolQuery.Where("name NOT IN ?", names)
		}
		if err := staleToolQuery.Pluck("id", &staleToolIDs).Error; err != nil {
			return err
		}
		if err := deleteConversationProjectMCPToolAssociations(tx, staleToolIDs); err != nil {
			return err
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

// deleteConversationProjectMCPToolAssociations 清理已删除工具的项目默认关联。
func deleteConversationProjectMCPToolAssociations(tx *gorm.DB, toolIDs []uint) error {
	if len(toolIDs) == 0 {
		return nil
	}
	return tx.Where("tool_id IN ?", toolIDs).Delete(&model.ConversationProjectMCPTool{}).Error
}

func (r *Repo) ListTools(ctx context.Context, serverID uint, onlyActive bool) ([]domainmcp.Tool, error) {
	query := r.db.WithContext(ctx).Where("server_id = ?", serverID).Order("sort_order asc").Order("name asc").Order("id asc")
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
	if err := r.db.WithContext(ctx).
		Joins("JOIN mcp_servers ON mcp_servers.id = mcp_tools.server_id").
		Where("mcp_tools.id IN ?", toolIDs).
		Order("mcp_servers.sort_order asc").
		Order("mcp_servers.id asc").
		Order("mcp_tools.sort_order asc").
		Order("mcp_tools.name asc").
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
	var result domainmcp.Tool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.MCPTool
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", toolID).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{}
		metadataChanged := false
		if input.DisplayName != nil && *input.DisplayName != row.DisplayName {
			updates["display_name"] = *input.DisplayName
			metadataChanged = true
		}
		if input.Description != nil && *input.Description != row.Description {
			updates["description"] = *input.Description
			metadataChanged = true
		}
		if input.AttachmentInputMode != nil && *input.AttachmentInputMode != row.AttachmentInputMode {
			updates["attachment_input_mode"] = *input.AttachmentInputMode
		}
		if input.AttachmentArgument != nil && *input.AttachmentArgument != row.AttachmentArgument {
			updates["attachment_argument"] = *input.AttachmentArgument
		}
		if input.AttachmentEncoding != nil && *input.AttachmentEncoding != row.AttachmentEncoding {
			updates["attachment_encoding"] = *input.AttachmentEncoding
		}
		if input.AttachmentPromptArgument != nil && *input.AttachmentPromptArgument != row.AttachmentPromptArgument {
			updates["attachment_prompt_argument"] = *input.AttachmentPromptArgument
		}
		if metadataChanged {
			updates["metadata_customized"] = true
		}
		if input.Status != nil && *input.Status != row.Status {
			updates["status"] = *input.Status
		}
		if input.DefaultEnabled != nil && *input.DefaultEnabled != row.DefaultEnabled {
			updates["default_enabled"] = *input.DefaultEnabled
		}
		if input.RequiresConfirm != nil && *input.RequiresConfirm != row.RequiresConfirm {
			updates["requires_confirm"] = *input.RequiresConfirm
		}
		if len(updates) > 0 {
			if err := tx.Model(&model.MCPTool{}).Where("id = ?", toolID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.First(&row, "id = ?", toolID).Error; err != nil {
				return err
			}
		}
		result = toDomainTool(row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
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

func (r *Repo) ReorderServersWithTools(ctx context.Context, order []repository.ReorderMCPServerInput) ([]domainmcp.ServerWithTools, error) {
	if len(order) == 0 {
		return []domainmcp.ServerWithTools{}, nil
	}
	returned := make([]domainmcp.ServerWithTools, 0)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		serverIDs := make([]uint, 0, len(order))
		for _, item := range order {
			serverIDs = append(serverIDs, item.ServerID)
		}
		var existingServers []model.MCPServer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", serverIDs).
			Find(&existingServers).Error; err != nil {
			return err
		}
		if len(existingServers) != len(order) {
			return gorm.ErrRecordNotFound
		}
		seenServers := make(map[uint]struct{}, len(order))
		for index, item := range order {
			if _, ok := seenServers[item.ServerID]; ok {
				return gorm.ErrRecordNotFound
			}
			seenServers[item.ServerID] = struct{}{}
			sortOrder := (index + 1) * 100
			if err := tx.Model(&model.MCPServer{}).
				Where("id = ?", item.ServerID).
				Update("sort_order", sortOrder).Error; err != nil {
				return err
			}
			var existingTools []model.MCPTool
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("server_id = ?", item.ServerID).
				Find(&existingTools).Error; err != nil {
				return err
			}
			if len(existingTools) != len(item.ToolIDs) {
				return gorm.ErrRecordNotFound
			}
			allowedTools := make(map[uint]struct{}, len(existingTools))
			for _, tool := range existingTools {
				allowedTools[tool.ID] = struct{}{}
			}
			seenTools := make(map[uint]struct{}, len(item.ToolIDs))
			for toolIndex, toolID := range item.ToolIDs {
				if _, ok := allowedTools[toolID]; !ok {
					return gorm.ErrRecordNotFound
				}
				if _, ok := seenTools[toolID]; ok {
					return gorm.ErrRecordNotFound
				}
				seenTools[toolID] = struct{}{}
				toolSortOrder := (toolIndex + 1) * 100
				if err := tx.Model(&model.MCPTool{}).
					Where("server_id = ? AND id = ?", item.ServerID, toolID).
					Update("sort_order", toolSortOrder).Error; err != nil {
					return err
				}
			}
		}

		servers, err := r.listServers(ctx, tx)
		if err != nil {
			return err
		}
		returned = make([]domainmcp.ServerWithTools, 0, len(servers))
		for _, server := range servers {
			var rows []model.MCPTool
			if err := tx.Where("server_id = ?", server.ID).
				Order("sort_order asc").
				Order("name asc").
				Order("id asc").
				Find(&rows).Error; err != nil {
				return err
			}
			tools := make([]domainmcp.Tool, 0, len(rows))
			for _, row := range rows {
				tool := toDomainTool(row)
				tool.ServerName = server.Name
				tools = append(tools, tool)
			}
			returned = append(returned, domainmcp.ServerWithTools{
				Server: server,
				Tools:  tools,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return returned, nil
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
		SortOrder:            row.SortOrder,
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
		ID:                       row.ID,
		ServerID:                 row.ServerID,
		Name:                     row.Name,
		DisplayName:              row.DisplayName,
		Description:              row.Description,
		InputSchemaJSON:          row.InputSchemaJSON,
		AttachmentInputMode:      row.AttachmentInputMode,
		AttachmentArgument:       row.AttachmentArgument,
		AttachmentEncoding:       row.AttachmentEncoding,
		AttachmentPromptArgument: row.AttachmentPromptArgument,
		Status:                   row.Status,
		SortOrder:                row.SortOrder,
		DefaultEnabled:           row.DefaultEnabled,
		RequiresConfirm:          row.RequiresConfirm,
		ToolKind:                 defaultToolKind(row.ToolKind),
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
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
