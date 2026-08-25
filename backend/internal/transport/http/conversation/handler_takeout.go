package conversation

import (
	"errors"
	"net/http"

	appconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// ExportConversationTakeout exports all conversations owned by the current user.
func (h *Handler) ExportConversationTakeout(c *gin.Context) {
	userID := middleware.MustUserID(c)

	item, err := h.service.ExportConversationTakeout(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "export conversations failed")
		return
	}

	h.recordAudit(c, "export_conversation_takeout",
		"conversation",
		"",
		map[string]interface{}{"conversation_count": item.TotalConversations, "message_count": item.TotalMessages},
	)

	response.Success(c, toConversationTakeoutPayload(item))
}

// ImportConversationTakeout imports a previously exported conversation JSON file.
func (h *Handler) ImportConversationTakeout(c *gin.Context) {
	userID := middleware.MustUserID(c)

	var req conversationTakeoutPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}

	result, err := h.service.ImportConversationTakeout(c.Request.Context(), userID, req.toApplication())
	if err != nil {
		if errors.Is(err, appconversation.ErrInvalidConversationImport) {
			response.Error(c, http.StatusBadRequest, "invalid conversation import")
			return
		}
		response.Error(c, http.StatusInternalServerError, "import conversations failed")
		return
	}

	h.recordAudit(c, "import_conversation_takeout",
		"conversation",
		"",
		map[string]interface{}{
			"conversation_count": result.ImportedConversationCount,
			"message_count":      result.ImportedMessageCount,
		},
	)

	response.Success(c, toConversationImportResponse(result))
}
