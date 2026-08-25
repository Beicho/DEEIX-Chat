package conversation

import (
	"context"
	"errors"
	"strings"

	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

const projectDocumentMaxBatchSize = 100

// ProjectDocumentInput 定义添加项目资料输入。
type ProjectDocumentInput struct {
	FileIDs []string
}

// ListProjectDocuments 查询当前用户项目资料。
func (s *Service) ListProjectDocuments(ctx context.Context, userID uint, projectPublicID string) ([]model.ProjectDocument, error) {
	items, err := s.repo.ListProjectDocuments(ctx, userID, strings.TrimSpace(projectPublicID))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrConversationProjectNotFound
	}
	return items, err
}

// AddProjectDocuments 将当前用户已有文件加入项目资料库。
func (s *Service) AddProjectDocuments(ctx context.Context, userID uint, projectPublicID string, input ProjectDocumentInput) ([]model.ProjectDocument, error) {
	fileIDs, err := normalizeProjectDocumentFileIDs(input.FileIDs)
	if err != nil {
		return nil, err
	}
	if _, err = s.repo.GetConversationProjectByPublicID(ctx, userID, strings.TrimSpace(projectPublicID)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrConversationProjectNotFound
		}
		return nil, err
	}
	items, err := s.repo.AddProjectDocuments(ctx, userID, strings.TrimSpace(projectPublicID), fileIDs)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, ErrFileNotFound
		default:
			return nil, err
		}
	}
	return items, nil
}

// DeleteProjectDocument 从项目资料库移除文件引用，不删除原文件。
func (s *Service) DeleteProjectDocument(ctx context.Context, userID uint, projectPublicID string, fileID string) error {
	if _, err := s.repo.GetConversationProjectByPublicID(ctx, userID, strings.TrimSpace(projectPublicID)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrConversationProjectNotFound
		}
		return err
	}
	err := s.repo.DeleteProjectDocument(ctx, userID, strings.TrimSpace(projectPublicID), strings.TrimSpace(fileID))
	if errors.Is(err, repository.ErrNotFound) {
		return ErrFileNotFound
	}
	return err
}

// ReindexProjectDocument 将资料索引状态标记为待建立，并同步触发文件索引重建能力。
func (s *Service) ReindexProjectDocument(ctx context.Context, userID uint, projectPublicID string, fileID string) (*model.ProjectDocument, error) {
	normalizedFileID := strings.TrimSpace(fileID)
	if normalizedFileID == "" {
		return nil, ErrInvalidFileReference
	}
	if _, err := s.repo.GetConversationProjectByPublicID(ctx, userID, strings.TrimSpace(projectPublicID)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrConversationProjectNotFound
		}
		return nil, err
	}
	if s.embeddingSvc != nil {
		fileObj, loadErr := s.repo.GetActiveFileObjectByID(ctx, userID, normalizedFileID)
		if errors.Is(loadErr, repository.ErrNotFound) || fileObj == nil {
			return nil, ErrFileNotFound
		}
		if loadErr != nil {
			return nil, loadErr
		}
		current, updateErr := s.repo.UpdateFileObjectEmbedStatus(
			ctx,
			userID,
			normalizedFileID,
			fileObj.EmbedSignature,
			"stale",
			"",
		)
		if updateErr != nil {
			return nil, updateErr
		}
		if current {
			fileObj.EmbedStatus = "stale"
			s.embeddingSvc.Trigger(*fileObj)
		}
	}
	item, err := s.repo.MarkProjectDocumentIndexStatus(ctx, userID, strings.TrimSpace(projectPublicID), normalizedFileID, "pending")
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFileNotFound
	}
	return item, err
}

// MarkProjectDocumentIndexStatus 设置资料索引状态，供索引管线回写。
func (s *Service) MarkProjectDocumentIndexStatus(ctx context.Context, userID uint, projectPublicID string, fileID string, indexStatus string) (*model.ProjectDocument, error) {
	status := normalizeProjectDocumentIndexStatus(indexStatus)
	if status == "" {
		return nil, ErrInvalidFileReference
	}
	if _, err := s.repo.GetConversationProjectByPublicID(ctx, userID, strings.TrimSpace(projectPublicID)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrConversationProjectNotFound
		}
		return nil, err
	}
	item, err := s.repo.MarkProjectDocumentIndexStatus(ctx, userID, strings.TrimSpace(projectPublicID), strings.TrimSpace(fileID), status)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFileNotFound
	}
	return item, err
}

func normalizeProjectDocumentFileIDs(values []string) ([]string, error) {
	if len(values) == 0 || len(values) > projectDocumentMaxBatchSize {
		return nil, ErrInvalidFileReference
	}
	results := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		fileID := strings.TrimSpace(value)
		if fileID == "" {
			continue
		}
		if _, ok := seen[fileID]; ok {
			continue
		}
		seen[fileID] = struct{}{}
		results = append(results, fileID)
	}
	if len(results) == 0 {
		return nil, ErrInvalidFileReference
	}
	return results, nil
}

func normalizeProjectDocumentIndexStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending", "indexing", "ready", "failed", "stale":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}
