package conversation

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	appconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// ArenaVoteRequest 竞技场投票请求。
type ArenaVoteRequest struct {
	MessageGroupID string `json:"messageGroupID" binding:"required,max=64"`
	WinnerModel    string `json:"winnerModel" binding:"required,max=128"`
	BlindMode      bool   `json:"blindMode"`
}

// ArenaVoteResponse 竞技场投票响应。
type ArenaVoteResponse struct {
	MessageGroupID string `json:"messageGroupID"`
	WinnerModel    string `json:"winnerModel"`
	BlindMode      bool   `json:"blindMode"`
	TotalVotes     int64  `json:"totalVotes"`
}

// ArenaLeaderboardEntryResponse 偏好榜单条目响应。
type ArenaLeaderboardEntryResponse struct {
	Model        string  `json:"model"`
	WinCount     int64   `json:"winCount"`
	TotalBattles int64   `json:"totalBattles"`
	WinRate      float64 `json:"winRate"`
}

// SubmitArenaVote godoc
// @Summary 竞技场投票
// @Description 对一组并排对比的模型分支投票，同组同用户只能投一次
// @Tags chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "会话 public_id"
// @Param body body ArenaVoteRequest true "投票参数"
// @Success 200 {object} response.SuccessDoc
// @Failure 400 {object} ErrorDoc
// @Failure 404 {object} ErrorDoc
// @Failure 409 {object} ErrorDoc
// @Router /conversations/{id}/arena-vote [post]
func (h *Handler) SubmitArenaVote(c *gin.Context) {
	userID := middleware.MustUserID(c)
	publicID, err := stringParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid conversation id")
		return
	}

	var req ArenaVoteRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}

	conversation, err := h.service.GetConversationByPublicID(c.Request.Context(), userID, publicID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "conversation not found")
		return
	}

	result, err := h.service.SubmitArenaVote(c.Request.Context(), appconversation.ArenaVoteInput{
		UserID:         userID,
		ConversationID: conversation.ID,
		MessageGroupID: req.MessageGroupID,
		WinnerModel:    strings.TrimSpace(req.WinnerModel),
		BlindMode:      req.BlindMode,
	})
	if err != nil {
		switch {
		case errors.Is(err, appconversation.ErrArenaInvalidVote):
			response.Error(c, http.StatusBadRequest, "invalid arena vote")
		case errors.Is(err, appconversation.ErrArenaVoteDuplicate):
			response.ErrorWithCode(c, http.StatusConflict, "arena_vote_duplicate", "you have already voted in this arena round")
		case errors.Is(err, appconversation.ErrConversationNotFound):
			response.Error(c, http.StatusNotFound, "conversation not found")
		default:
			response.Error(c, http.StatusInternalServerError, "submit arena vote failed")
		}
		return
	}

	h.recordAudit(c, "arena_vote", "conversation", publicID, map[string]interface{}{
		"messageGroupID": result.MessageGroupID,
		"winnerModel":    result.WinnerModel,
		"blindMode":      result.BlindMode,
	})

	response.Success(c, ArenaVoteResponse{
		MessageGroupID: result.MessageGroupID,
		WinnerModel:    result.WinnerModel,
		BlindMode:      result.BlindMode,
		TotalVotes:     result.TotalVotes,
	})
}

// GetArenaLeaderboard godoc
// @Summary 竞技场偏好榜
// @Description 聚合各模型的胜场、总对局数与胜率，按胜率降序
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param limit query int false "返回条数上限"
// @Success 200 {object} response.SuccessDoc
// @Router /admin/arena/leaderboard [get]
func (h *Handler) GetArenaLeaderboard(c *gin.Context) {
	limit := 0
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}

	entries, err := h.service.GetArenaLeaderboard(c.Request.Context(), limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "load arena leaderboard failed")
		return
	}

	items := make([]ArenaLeaderboardEntryResponse, 0, len(entries))
	for _, entry := range entries {
		items = append(items, ArenaLeaderboardEntryResponse{
			Model:        entry.Model,
			WinCount:     entry.WinCount,
			TotalBattles: entry.TotalBattles,
			WinRate:      entry.WinRate,
		})
	}

	response.Success(c, gin.H{"leaderboard": items})
}
