package mcp

import "github.com/gin-gonic/gin"

func (m *Module) RegisterRoutes(authGroup *gin.RouterGroup) {
	group := authGroup.Group("/mcp")
	group.GET("/tools", m.Handler.ListAvailableTools)
	group.GET("/servers", m.Handler.ListUserServers)
	group.POST("/servers", m.Handler.CreateUserServer)
	group.PATCH("/servers/:id", m.Handler.UpdateUserServer)
	group.DELETE("/servers/:id", m.Handler.DeleteUserServer)
	group.GET("/servers/:id/tools", m.Handler.ListUserServerTools)
	group.POST("/servers/:id/sync", m.Handler.SyncUserServerTools)
	group.POST("/servers/:id/test", m.Handler.TestUserServerConnection)
	group.POST("/servers/:id/oauth/start", m.Handler.StartOAuth)
	group.POST("/oauth/callback", m.Handler.CompleteOAuthCallback)
	group.GET("/tool-selection", m.Handler.GetToolPreference)
	group.PUT("/tool-selection", m.Handler.PutToolPreference)

	voiceGroup := authGroup.Group("/voice")
	voiceGroup.GET("/config", m.Handler.GetVoiceConfig)
	voiceGroup.POST("/asr", m.Handler.ASR)
	voiceGroup.POST("/tts", m.Handler.TTS)
}

func (m *Module) RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	group := adminGroup.Group("/mcp")
	group.GET("/servers", m.Handler.ListServers)
	group.POST("/servers", m.Handler.CreateServer)
	group.PATCH("/servers/order", m.Handler.ReorderServers)
	group.PATCH("/servers/:id", m.Handler.UpdateServer)
	group.DELETE("/servers/:id", m.Handler.DeleteServer)
	group.GET("/servers/:id/tools", m.Handler.ListServerTools)
	group.PATCH("/servers/:id/tools/status", m.Handler.UpdateServerToolsStatus)
	group.POST("/servers/:id/test", m.Handler.TestServerConnection)
	group.POST("/servers/:id/sync", m.Handler.SyncServerTools)
	group.PATCH("/tools/:id", m.Handler.UpdateTool)
}
