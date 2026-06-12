package collaboration

import "github.com/gin-gonic/gin"

// RegisterRoutes registers user collaboration routes.
func (m *Module) RegisterRoutes(authRequired *gin.RouterGroup) {
	authRequired.GET("/assistants", m.Handler.ListAssistants)
	authRequired.POST("/assistants", m.Handler.CreateAssistant)
	authRequired.PUT("/assistants/:id", m.Handler.UpdateAssistant)
	authRequired.DELETE("/assistants/:id", m.Handler.DeleteAssistant)
	authRequired.GET("/assistant-marketplace", m.Handler.ListMarketplaceAssistants)
	authRequired.POST("/assistant-marketplace/:id/install", m.Handler.InstallAssistant)
	authRequired.DELETE("/assistant-marketplace/:id/install", m.Handler.UninstallAssistant)

	authRequired.GET("/scheduled-prompts", m.Handler.ListScheduledPrompts)
	authRequired.POST("/scheduled-prompts", m.Handler.CreateScheduledPrompt)
	authRequired.PUT("/scheduled-prompts/:id", m.Handler.UpdateScheduledPrompt)
	authRequired.DELETE("/scheduled-prompts/:id", m.Handler.DeleteScheduledPrompt)

	authRequired.GET("/team-spaces", m.Handler.ListTeamSpaces)
	authRequired.POST("/team-spaces", m.Handler.CreateTeamSpace)
	authRequired.POST("/team-spaces/:id/members", m.Handler.AddTeamMember)
	authRequired.DELETE("/team-spaces/:id/members/:user_id", m.Handler.RemoveTeamMember)
}
