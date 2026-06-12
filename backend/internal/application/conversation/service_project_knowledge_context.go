package conversation

import (
	"context"
	"strings"

	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
)

func (s *Service) projectDocumentFileIDsForConversation(ctx context.Context, userID uint, conversation *model.Conversation) []string {
	if s == nil || s.repo == nil || conversation == nil || conversation.ProjectID == nil || *conversation.ProjectID == 0 {
		return nil
	}
	files, err := s.repo.ListProjectDocumentFilesByProjectID(ctx, userID, *conversation.ProjectID)
	if err != nil || len(files) == 0 {
		return nil
	}
	result := make([]string, 0, len(files))
	for _, file := range files {
		if fileID := strings.TrimSpace(file.FileID); fileID != "" {
			result = append(result, fileID)
		}
	}
	return result
}

func mergeFileIDs(groups ...[]string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, group := range groups {
		for _, raw := range group {
			fileID := strings.TrimSpace(raw)
			if fileID == "" {
				continue
			}
			if _, ok := seen[fileID]; ok {
				continue
			}
			seen[fileID] = struct{}{}
			result = append(result, fileID)
		}
	}
	return result
}
