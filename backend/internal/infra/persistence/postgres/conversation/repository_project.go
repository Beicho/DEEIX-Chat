package conversation

import (
	"context"
	"strings"

	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	domainknowledgebase "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/knowledgebase"
	models "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// maxConversationProjectsPerUser 单用户会话项目数量上限；列表接口为全量加载，需要写入侧兜底有界。
const maxConversationProjectsPerUser = 200

// CreateConversationProject 创建会话项目分组。
func (r *Repo) CreateConversationProject(ctx context.Context, item *domainconversation.ConversationProject) error {
	entity := toConversationProjectModel(item)
	mcpToolIDs := append([]uint(nil), item.DefaultMCPToolIDs...)
	skillIDs := append([]uint(nil), item.DefaultSkillIDs...)
	knowledgeBaseIDs := append([]string(nil), item.DefaultKnowledgeBaseIDs...)
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.ConversationProject{}).
			Where("user_id = ?", entity.UserID).
			Count(&count).Error; err != nil {
			return err
		}
		if count >= maxConversationProjectsPerUser {
			return repository.ErrConversationProjectLimitExceeded
		}
		if err := tx.Create(&entity).Error; err != nil {
			return err
		}
		if err := replaceConversationProjectMCPTools(tx, entity.ID, mcpToolIDs); err != nil {
			return err
		}
		if err := replaceConversationProjectSkills(tx, entity.ID, skillIDs); err != nil {
			return err
		}
		return replaceConversationProjectKnowledgeBases(tx, entity.ID, entity.UserID, knowledgeBaseIDs)
	}); err != nil {
		return translateError(err)
	}
	*item = toConversationProjectDomain(entity)
	item.DefaultMCPToolIDs = mcpToolIDs
	item.DefaultSkillIDs = skillIDs
	item.DefaultKnowledgeBaseIDs = knowledgeBaseIDs
	return nil
}

// ListConversationProjects 查询用户项目分组。
func (r *Repo) ListConversationProjects(ctx context.Context, userID uint, statusFilter string) ([]domainconversation.ConversationProject, error) {
	items := make([]models.ConversationProject, 0)
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID)
	switch strings.TrimSpace(statusFilter) {
	case "archived":
		query = query.Where("status = ?", "archived")
	case "all":
		// 保留全部状态。
	default:
		query = query.Where("status = ?", "active")
	}
	if err := query.
		Order("sort_order ASC").
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, translateError(err)
	}
	results := toConversationProjectDomains(items)
	if err := r.hydrateConversationProjectDefaults(ctx, results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetConversationProjectByPublicID 查询用户项目分组。
func (r *Repo) GetConversationProjectByPublicID(ctx context.Context, userID uint, publicID string) (*domainconversation.ConversationProject, error) {
	var item models.ConversationProject
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND public_id = ?", userID, strings.TrimSpace(publicID)).
		First(&item).Error; err != nil {
		return nil, translateError(err)
	}
	result := toConversationProjectDomain(item)
	projects := []domainconversation.ConversationProject{result}
	if err := r.hydrateConversationProjectDefaults(ctx, projects); err != nil {
		return nil, err
	}
	result = projects[0]
	return &result, nil
}

// UpdateConversationProjectMetadataByPublicID 更新项目分组元信息。
func (r *Repo) UpdateConversationProjectMetadataByPublicID(
	ctx context.Context,
	userID uint,
	publicID string,
	patch domainconversation.ConversationProjectPatch,
) (*domainconversation.ConversationProject, error) {
	updates := make(map[string]interface{})
	if patch.Name != nil {
		updates["name"] = *patch.Name
	}
	if patch.Description != nil {
		updates["description"] = *patch.Description
	}
	if patch.SystemPrompt != nil {
		updates["system_prompt"] = *patch.SystemPrompt
	}
	if patch.MCPDefaultMode != nil {
		updates["mcp_default_mode"] = *patch.MCPDefaultMode
	}
	if patch.Color != nil {
		updates["color"] = *patch.Color
	}
	if patch.Icon != nil {
		updates["icon"] = *patch.Icon
	}
	if patch.Status != nil {
		updates["status"] = *patch.Status
	}
	if len(updates) == 0 && patch.DefaultMCPToolIDs == nil && patch.DefaultSkillIDs == nil && patch.DefaultKnowledgeBaseIDs == nil {
		return r.GetConversationProjectByPublicID(ctx, userID, publicID)
	}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project models.ConversationProject
		if err := tx.Where("user_id = ? AND public_id = ?", userID, strings.TrimSpace(publicID)).First(&project).Error; err != nil {
			return err
		}
		if len(updates) > 0 {
			if err := tx.Model(&project).Updates(updates).Error; err != nil {
				return err
			}
		}
		if patch.DefaultMCPToolIDs != nil {
			if err := replaceConversationProjectMCPTools(tx, project.ID, *patch.DefaultMCPToolIDs); err != nil {
				return err
			}
		}
		if patch.DefaultSkillIDs != nil {
			if err := replaceConversationProjectSkills(tx, project.ID, *patch.DefaultSkillIDs); err != nil {
				return err
			}
		}
		if patch.DefaultKnowledgeBaseIDs != nil {
			if err := replaceConversationProjectKnowledgeBases(tx, project.ID, project.UserID, *patch.DefaultKnowledgeBaseIDs); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, translateError(err)
	}
	return r.GetConversationProjectByPublicID(ctx, userID, publicID)
}

// DeleteConversationProjectByPublicID 删除项目分组，可选择一并软删除其下会话并返回可清理文件 ID。
func (r *Repo) DeleteConversationProjectByPublicID(
	ctx context.Context,
	userID uint,
	publicID string,
	deleteConversations bool,
	deleteFiles bool,
) ([]string, error) {
	normalizedPublicID := strings.TrimSpace(publicID)
	cleanupFileIDs := make([]string, 0)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project models.ConversationProject
		if err := tx.Where("user_id = ? AND public_id = ?", userID, normalizedPublicID).First(&project).Error; err != nil {
			return translateError(err)
		}
		// 项目删除与会话归属处理必须保持原子性，避免项目删除后留下不可见的项目引用。
		if deleteConversations {
			conversationIDs := make([]uint, 0)
			if deleteFiles {
				if err := tx.Model(&models.Conversation{}).
					Where("user_id = ? AND project_id = ?", userID, project.ID).
					Pluck("id", &conversationIDs).Error; err != nil {
					return translateError(err)
				}
			}
			if err := tx.
				Where("user_id = ? AND project_id = ?", userID, project.ID).
				Delete(&models.Conversation{}).Error; err != nil {
				return translateError(err)
			}
			if deleteFiles {
				fileIDs, err := listConversationFileCleanupCandidates(tx, userID, conversationIDs)
				if err != nil {
					return err
				}
				cleanupFileIDs = fileIDs
			}
		} else {
			if err := tx.Model(&models.Conversation{}).
				Where("user_id = ? AND project_id = ?", userID, project.ID).
				Update("project_id", nil).Error; err != nil {
				return translateError(err)
			}
		}
		if err := tx.Where("project_id = ?", project.ID).Delete(&models.ConversationProjectMCPTool{}).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Where("project_id = ?", project.ID).Delete(&models.ConversationProjectSkill{}).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Where("project_id = ?", project.ID).Delete(&models.ConversationProjectKnowledgeBase{}).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Delete(&project).Error; err != nil {
			return translateError(err)
		}
		return nil
	})
	if err != nil {
		return nil, translateError(err)
	}
	return cleanupFileIDs, nil
}

// ReorderConversationProjects 更新项目展示顺序。
func (r *Repo) ReorderConversationProjects(ctx context.Context, userID uint, publicIDs []string) error {
	if len(publicIDs) == 0 {
		return nil
	}
	return translateError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index, publicID := range publicIDs {
			result := tx.Model(&models.ConversationProject{}).
				Where("user_id = ? AND public_id = ?", userID, strings.TrimSpace(publicID)).
				Update("sort_order", index+1)
			if result.Error != nil {
				return translateError(result.Error)
			}
			if result.RowsAffected == 0 {
				return repository.ErrNotFound
			}
		}
		return nil
	}))
}

// UpdateConversationProjectAssignmentByPublicID 更新单个会话的项目归属。
func (r *Repo) UpdateConversationProjectAssignmentByPublicID(
	ctx context.Context,
	userID uint,
	conversationPublicID string,
	projectID *uint,
) (*domainconversation.Conversation, error) {
	result := r.db.WithContext(ctx).
		Model(&models.Conversation{}).
		Where("user_id = ? AND public_id = ?", userID, strings.TrimSpace(conversationPublicID)).
		Update("project_id", projectID)
	if result.Error != nil {
		return nil, translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	return r.GetConversationByPublicID(ctx, conversationPublicID, userID)
}

// BatchUpdateConversationProjectByPublicIDs 批量更新会话项目归属。
func (r *Repo) BatchUpdateConversationProjectByPublicIDs(
	ctx context.Context,
	userID uint,
	conversationPublicIDs []string,
	projectID *uint,
) (int64, error) {
	if len(conversationPublicIDs) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.Conversation{}).
		Where("user_id = ? AND public_id IN ?", userID, conversationPublicIDs).
		Update("project_id", projectID)
	if result.Error != nil {
		return 0, translateError(result.Error)
	}
	return result.RowsAffected, nil
}

// ListProjectDocuments 查询当前用户指定项目的资料。
func (r *Repo) ListProjectDocuments(ctx context.Context, userID uint, projectPublicID string) ([]domainconversation.ProjectDocument, error) {
	project, err := r.GetConversationProjectByPublicID(ctx, userID, projectPublicID)
	if err != nil {
		return nil, err
	}
	return r.listProjectDocumentsByProjectID(ctx, userID, project.ID)
}

// AddProjectDocuments 添加当前用户指定项目的资料引用。
func (r *Repo) AddProjectDocuments(ctx context.Context, userID uint, projectPublicID string, fileIDs []string) ([]domainconversation.ProjectDocument, error) {
	if len(fileIDs) == 0 {
		return []domainconversation.ProjectDocument{}, nil
	}
	project, err := r.GetConversationProjectByPublicID(ctx, userID, projectPublicID)
	if err != nil {
		return nil, err
	}
	files := make([]models.FileObject, 0, len(fileIDs))
	if err = r.db.WithContext(ctx).
		Where("user_id = ? AND status = ? AND file_id IN ?", userID, "active", fileIDs).
		Find(&files).Error; err != nil {
		return nil, translateError(err)
	}
	if len(files) != len(fileIDs) {
		return nil, repository.ErrNotFound
	}
	fileByID := make(map[string]models.FileObject, len(files))
	for _, file := range files {
		fileByID[file.FileID] = file
	}
	rows := make([]models.ProjectDocument, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		file := fileByID[strings.TrimSpace(fileID)]
		rows = append(rows, models.ProjectDocument{
			UserID:      userID,
			ProjectID:   project.ID,
			FileObjID:   file.ID,
			FileID:      file.FileID,
			IndexStatus: projectDocumentIndexStatusFromFile(file),
			Status:      "active",
		})
	}
	if err = r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "project_id"}, {Name: "file_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"file_obj_id":  gorm.Expr("excluded.file_obj_id"),
				"index_status": gorm.Expr("excluded.index_status"),
				"status":       "active",
				"deleted_at":   nil,
			}),
		}).
		Create(&rows).Error; err != nil {
		return nil, translateError(err)
	}
	return r.listProjectDocumentsByProjectID(ctx, userID, project.ID)
}

// DeleteProjectDocument 从项目资料库移除一个文件引用。
func (r *Repo) DeleteProjectDocument(ctx context.Context, userID uint, projectPublicID string, fileID string) error {
	project, err := r.GetConversationProjectByPublicID(ctx, userID, projectPublicID)
	if err != nil {
		return err
	}
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND project_id = ? AND file_id = ?", userID, project.ID, strings.TrimSpace(fileID)).
		Delete(&models.ProjectDocument{})
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// MarkProjectDocumentIndexStatus 标记项目资料索引状态。
func (r *Repo) MarkProjectDocumentIndexStatus(ctx context.Context, userID uint, projectPublicID string, fileID string, indexStatus string) (*domainconversation.ProjectDocument, error) {
	project, err := r.GetConversationProjectByPublicID(ctx, userID, projectPublicID)
	if err != nil {
		return nil, err
	}
	result := r.db.WithContext(ctx).
		Model(&models.ProjectDocument{}).
		Where("user_id = ? AND project_id = ? AND file_id = ?", userID, project.ID, strings.TrimSpace(fileID)).
		Update("index_status", strings.TrimSpace(indexStatus))
	if result.Error != nil {
		return nil, translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	items, err := r.listProjectDocumentsByProjectID(ctx, userID, project.ID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].FileID == strings.TrimSpace(fileID) {
			return &items[i], nil
		}
	}
	return nil, repository.ErrNotFound
}

// ListProjectDocumentFilesByProjectID 查询会话所属项目可用于上下文的资料文件。
func (r *Repo) ListProjectDocumentFilesByProjectID(ctx context.Context, userID uint, projectID uint) ([]domainconversation.FileObject, error) {
	if projectID == 0 {
		return []domainconversation.FileObject{}, nil
	}
	files := make([]models.FileObject, 0)
	if err := r.db.WithContext(ctx).
		Table("file_objects").
		Select("file_objects.*").
		Joins("JOIN project_documents ON project_documents.file_obj_id = file_objects.id").
		Where("project_documents.user_id = ? AND project_documents.project_id = ? AND project_documents.deleted_at IS NULL", userID, projectID).
		Where("project_documents.status = ? AND file_objects.status = ?", "active", "active").
		Order("project_documents.id ASC").
		Find(&files).Error; err != nil {
		return nil, translateError(err)
	}
	return toFileObjectDomains(files), nil
}

func (r *Repo) listProjectDocumentsByProjectID(ctx context.Context, userID uint, projectID uint) ([]domainconversation.ProjectDocument, error) {
	type row struct {
		models.ProjectDocument
		FileName      string `gorm:"column:file_name"`
		FileSize      int64  `gorm:"column:size_bytes"`
		FileCategory  string `gorm:"column:file_category"`
		ExtractStatus string `gorm:"column:extract_status"`
		EmbedStatus   string `gorm:"column:embed_status"`
	}
	rows := make([]row, 0)
	if err := r.db.WithContext(ctx).
		Table("project_documents").
		Select("project_documents.*, file_objects.file_name, file_objects.size_bytes, file_objects.file_category, file_objects.extract_status, file_objects.embed_status").
		Joins("JOIN file_objects ON file_objects.id = project_documents.file_obj_id").
		Where("project_documents.deleted_at IS NULL").
		Where("project_documents.user_id = ? AND project_documents.project_id = ? AND project_documents.status = ? AND file_objects.status = ?", userID, projectID, "active", "active").
		Order("project_documents.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, translateError(err)
	}
	results := make([]domainconversation.ProjectDocument, 0, len(rows))
	for _, item := range rows {
		results = append(results, domainconversation.ProjectDocument{
			ID:            item.ID,
			UserID:        item.UserID,
			ProjectID:     item.ProjectID,
			FileObjID:     item.FileObjID,
			FileID:        item.FileID,
			FileName:      item.FileName,
			FileSize:      item.FileSize,
			FileCategory:  item.FileCategory,
			ExtractStatus: item.ExtractStatus,
			EmbedStatus:   item.EmbedStatus,
			IndexStatus:   item.IndexStatus,
			Status:        item.Status,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	return results, nil
}

func projectDocumentIndexStatusFromFile(file models.FileObject) string {
	switch strings.TrimSpace(file.EmbedStatus) {
	case "ready":
		return "ready"
	case "failed":
		return "failed"
	case "processing":
		return "indexing"
	default:
		if strings.TrimSpace(file.ExtractStatus) == "ready" {
			return "ready"
		}
		return "pending"
	}
}

func toConversationProjectDomain(item models.ConversationProject) domainconversation.ConversationProject {
	mcpDefaultMode := strings.TrimSpace(item.MCPDefaultMode)
	if mcpDefaultMode != domainconversation.ConversationProjectMCPDefaultModeCustom {
		mcpDefaultMode = domainconversation.ConversationProjectMCPDefaultModeInherit
	}
	return domainconversation.ConversationProject{
		ID:             item.ID,
		UserID:         item.UserID,
		PublicID:       item.PublicID,
		Name:           item.Name,
		Description:    item.Description,
		SystemPrompt:   item.SystemPrompt,
		MCPDefaultMode: mcpDefaultMode,
		Color:          item.Color,
		Icon:           item.Icon,
		SortOrder:      item.SortOrder,
		Status:         item.Status,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func toConversationProjectDomains(items []models.ConversationProject) []domainconversation.ConversationProject {
	results := make([]domainconversation.ConversationProject, 0, len(items))
	for _, item := range items {
		results = append(results, toConversationProjectDomain(item))
	}
	return results
}

func toConversationProjectModel(item *domainconversation.ConversationProject) models.ConversationProject {
	if item == nil {
		return models.ConversationProject{}
	}
	return models.ConversationProject{
		UserID:         item.UserID,
		PublicID:       item.PublicID,
		Name:           item.Name,
		Description:    item.Description,
		SystemPrompt:   item.SystemPrompt,
		MCPDefaultMode: item.MCPDefaultMode,
		Color:          item.Color,
		Icon:           item.Icon,
		SortOrder:      item.SortOrder,
		Status:         item.Status,
	}
}

// hydrateConversationProjectDefaults 批量装载项目默认 MCP 与 Skill 关联，避免列表查询产生 N+1。
func (r *Repo) hydrateConversationProjectDefaults(ctx context.Context, items []domainconversation.ConversationProject) error {
	if len(items) == 0 {
		return nil
	}
	projectIDs := make([]uint, 0, len(items))
	for _, item := range items {
		projectIDs = append(projectIDs, item.ID)
	}

	mcpRows := make([]models.ConversationProjectMCPTool, 0)
	if err := r.db.WithContext(ctx).
		Where("project_id IN ?", projectIDs).
		Order("project_id ASC, sort_order ASC, tool_id ASC").
		Find(&mcpRows).Error; err != nil {
		return translateError(err)
	}
	skillRows := make([]models.ConversationProjectSkill, 0)
	if err := r.db.WithContext(ctx).
		Where("project_id IN ?", projectIDs).
		Order("project_id ASC, sort_order ASC, skill_id ASC").
		Find(&skillRows).Error; err != nil {
		return translateError(err)
	}
	knowledgeBaseRows := make([]struct {
		ProjectID uint
		PublicID  string
	}, 0)
	if err := r.db.WithContext(ctx).Table("chat_conversation_project_knowledge_bases AS project_bases").
		Select("project_bases.project_id, knowledge_bases.public_id").
		Joins("JOIN knowledge_bases ON knowledge_bases.id = project_bases.knowledge_base_id").
		Joins("JOIN chat_conversation_projects AS projects ON projects.id = project_bases.project_id").
		Where("project_bases.project_id IN ?", projectIDs).
		Where("knowledge_bases.enabled = ?", true).
		Where("knowledge_bases.scope = ? OR (knowledge_bases.scope = ? AND knowledge_bases.owner_user_id = projects.user_id)",
			domainknowledgebase.ScopeBuiltin, domainknowledgebase.ScopeUser).
		Order("project_bases.project_id ASC, project_bases.sort_order ASC, project_bases.knowledge_base_id ASC").
		Scan(&knowledgeBaseRows).Error; err != nil {
		return translateError(err)
	}

	mcpIDsByProject := make(map[uint][]uint, len(items))
	for _, row := range mcpRows {
		mcpIDsByProject[row.ProjectID] = append(mcpIDsByProject[row.ProjectID], row.ToolID)
	}
	skillIDsByProject := make(map[uint][]uint, len(items))
	for _, row := range skillRows {
		skillIDsByProject[row.ProjectID] = append(skillIDsByProject[row.ProjectID], row.SkillID)
	}
	knowledgeBaseIDsByProject := make(map[uint][]string, len(items))
	for _, row := range knowledgeBaseRows {
		knowledgeBaseIDsByProject[row.ProjectID] = append(knowledgeBaseIDsByProject[row.ProjectID], row.PublicID)
	}
	for index := range items {
		items[index].DefaultMCPToolIDs = mcpIDsByProject[items[index].ID]
		items[index].DefaultSkillIDs = skillIDsByProject[items[index].ID]
		items[index].DefaultKnowledgeBaseIDs = knowledgeBaseIDsByProject[items[index].ID]
	}
	return nil
}

func replaceConversationProjectKnowledgeBases(tx *gorm.DB, projectID uint, userID uint, publicIDs []string) error {
	if err := tx.Where("project_id = ?", projectID).Delete(&models.ConversationProjectKnowledgeBase{}).Error; err != nil {
		return err
	}
	if len(publicIDs) == 0 {
		return nil
	}
	bases := make([]models.KnowledgeBase, 0, len(publicIDs))
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("public_id IN ? AND enabled = ?", publicIDs, true).
		Where("scope = ? OR (scope = ? AND owner_user_id = ?)", domainknowledgebase.ScopeBuiltin, domainknowledgebase.ScopeUser, userID).
		Find(&bases).Error; err != nil {
		return err
	}
	baseIDByPublicID := make(map[string]uint, len(bases))
	for _, base := range bases {
		baseIDByPublicID[base.PublicID] = base.ID
	}
	if len(baseIDByPublicID) != len(publicIDs) {
		return repository.ErrNotFound
	}
	rows := make([]models.ConversationProjectKnowledgeBase, 0, len(publicIDs))
	for index, publicID := range publicIDs {
		rows = append(rows, models.ConversationProjectKnowledgeBase{ProjectID: projectID, KnowledgeBaseID: baseIDByPublicID[publicID], SortOrder: index + 1})
	}
	return tx.Create(&rows).Error
}

// replaceConversationProjectMCPTools 在事务内替换项目默认 MCP 工具关联。
func replaceConversationProjectMCPTools(tx *gorm.DB, projectID uint, toolIDs []uint) error {
	if err := tx.Where("project_id = ?", projectID).Delete(&models.ConversationProjectMCPTool{}).Error; err != nil {
		return err
	}
	rows := make([]models.ConversationProjectMCPTool, 0, len(toolIDs))
	for index, toolID := range toolIDs {
		rows = append(rows, models.ConversationProjectMCPTool{ProjectID: projectID, ToolID: toolID, SortOrder: index + 1})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// replaceConversationProjectSkills 在事务内替换项目默认 Skill 关联。
func replaceConversationProjectSkills(tx *gorm.DB, projectID uint, skillIDs []uint) error {
	if err := tx.Where("project_id = ?", projectID).Delete(&models.ConversationProjectSkill{}).Error; err != nil {
		return err
	}
	rows := make([]models.ConversationProjectSkill, 0, len(skillIDs))
	for index, skillID := range skillIDs {
		rows = append(rows, models.ConversationProjectSkill{ProjectID: projectID, SkillID: skillID, SortOrder: index + 1})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}
