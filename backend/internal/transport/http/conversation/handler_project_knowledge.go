package conversation

import (
	"errors"
	"net/http"

	appconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ListProjectDocuments(c *gin.Context) {
	userID := middleware.MustUserID(c)
	projectID, err := stringParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid conversation project id")
		return
	}
	items, err := h.service.ListProjectDocuments(c.Request.Context(), userID, projectID)
	if err != nil {
		writeProjectKnowledgeError(c, err, "list project documents failed")
		return
	}
	response.Success(c, toProjectDocumentResponses(items))
}

func (h *Handler) AddProjectDocuments(c *gin.Context) {
	userID := middleware.MustUserID(c)
	projectID, err := stringParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid conversation project id")
		return
	}
	var req AddProjectDocumentsRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	items, err := h.service.AddProjectDocuments(c.Request.Context(), userID, projectID, appconversation.ProjectDocumentInput{
		FileIDs: req.FileIDs,
	})
	if err != nil {
		writeProjectKnowledgeError(c, err, "add project documents failed")
		return
	}
	h.recordAudit(c, "add_project_documents", "conversation_project", projectID, map[string]int{"count": len(req.FileIDs)})
	response.Success(c, toProjectDocumentResponses(items))
}

func (h *Handler) DeleteProjectDocument(c *gin.Context) {
	userID := middleware.MustUserID(c)
	projectID, err := stringParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid conversation project id")
		return
	}
	fileID, err := stringParam(c, "file_id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid file id")
		return
	}
	if err = h.service.DeleteProjectDocument(c.Request.Context(), userID, projectID, fileID); err != nil {
		writeProjectKnowledgeError(c, err, "delete project document failed")
		return
	}
	h.recordAudit(c, "delete_project_document", "conversation_project", projectID, map[string]string{"fileID": fileID})
	response.Success(c, gin.H{"deleted": true, "fileID": fileID})
}

func (h *Handler) ReindexProjectDocument(c *gin.Context) {
	userID := middleware.MustUserID(c)
	projectID, err := stringParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid conversation project id")
		return
	}
	fileID, err := stringParam(c, "file_id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid file id")
		return
	}
	item, err := h.service.ReindexProjectDocument(c.Request.Context(), userID, projectID, fileID)
	if err != nil {
		writeProjectKnowledgeError(c, err, "reindex project document failed")
		return
	}
	response.Success(c, toProjectDocumentResponse(*item))
}

func (h *Handler) MarkProjectDocumentIndexStatus(c *gin.Context) {
	userID := middleware.MustUserID(c)
	projectID, err := stringParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid conversation project id")
		return
	}
	fileID, err := stringParam(c, "file_id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid file id")
		return
	}
	var req MarkProjectDocumentIndexRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	item, err := h.service.MarkProjectDocumentIndexStatus(c.Request.Context(), userID, projectID, fileID, req.IndexStatus)
	if err != nil {
		writeProjectKnowledgeError(c, err, "mark project document index failed")
		return
	}
	response.Success(c, toProjectDocumentResponse(*item))
}

func writeProjectKnowledgeError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, appconversation.ErrConversationProjectNotFound):
		response.Error(c, http.StatusNotFound, "conversation project not found")
	case errors.Is(err, appconversation.ErrFileNotFound):
		response.Error(c, http.StatusNotFound, "file not found")
	case errors.Is(err, appconversation.ErrInvalidFileReference):
		response.Error(c, http.StatusBadRequest, "invalid file reference")
	default:
		response.Error(c, http.StatusInternalServerError, fallback)
	}
}
