package collaboration

import "errors"

var (
	ErrAssistantNotFound       = errors.New("assistant not found")
	ErrScheduledPromptNotFound = errors.New("scheduled prompt not found")
	ErrTeamSpaceNotFound       = errors.New("team space not found")
	ErrTeamMemberNotFound      = errors.New("team member not found")
	ErrUserNotFound            = errors.New("user not found")
	ErrInvalidInput            = errors.New("invalid input")
	ErrContentBlocked          = errors.New("content blocked")
)
