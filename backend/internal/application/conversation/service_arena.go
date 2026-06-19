package conversation

import (
	"context"
	"errors"
	"strings"

	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

// ArenaVoteInput 竞技场投票请求参数。
type ArenaVoteInput struct {
	UserID         uint
	ConversationID uint
	MessageGroupID string
	WinnerModel    string
	BlindMode      bool
}

// ArenaVoteResult 竞技场投票结果。
type ArenaVoteResult struct {
	MessageGroupID string
	WinnerModel    string
	BlindMode      bool
	TotalVotes     int64
}

// SubmitArenaVote 记录一次竞技场偏好投票，同组同用户重复投票返回 ErrArenaVoteDuplicate。
func (s *Service) SubmitArenaVote(ctx context.Context, input ArenaVoteInput) (*ArenaVoteResult, error) {
	groupID := normalizePublicID(input.MessageGroupID)
	if groupID == "" {
		return nil, ErrArenaInvalidVote
	}
	winnerModel := strings.TrimSpace(input.WinnerModel)
	if winnerModel == "" {
		return nil, ErrArenaInvalidVote
	}

	// 归属校验：会话必须属于当前用户，避免越权对他人会话投票。
	if _, err := s.repo.GetConversationByUser(ctx, input.ConversationID, input.UserID); err != nil {
		return nil, ErrConversationNotFound
	}

	vote := &model.ArenaVote{
		MessageGroupID: groupID,
		ConversationID: input.ConversationID,
		VoterUserID:    input.UserID,
		WinnerModel:    winnerModel,
		BlindMode:      input.BlindMode,
	}
	if err := s.repo.CreateArenaVote(ctx, vote); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrArenaVoteDuplicate
		}
		return nil, err
	}

	total, err := s.repo.CountArenaVotesByGroup(ctx, groupID)
	if err != nil {
		// 投票已落库，计数失败不影响幂等结果，降级返回 0。
		total = 0
	}

	return &ArenaVoteResult{
		MessageGroupID: groupID,
		WinnerModel:    winnerModel,
		BlindMode:      input.BlindMode,
		TotalVotes:     total,
	}, nil
}

// GetArenaLeaderboard 返回模型偏好榜，按胜率降序。limit<=0 时返回全部。
func (s *Service) GetArenaLeaderboard(ctx context.Context, limit int) ([]model.ArenaLeaderboardEntry, error) {
	return s.repo.ListArenaLeaderboard(ctx, limit)
}
