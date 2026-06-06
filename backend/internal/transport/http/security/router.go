package security

import "github.com/gin-gonic/gin"

func (m *Module) RegisterRoutes(authRequired *gin.RouterGroup) {
	group := authRequired.Group("/security")
	group.POST("/pow/challenge", m.Handler.GetPoWChallenge)
	group.POST("/browser-key/bootstrap", m.Handler.BootstrapBrowserKey)
	group.POST("/fingerprint", m.Handler.RecordFingerprint)
}

func (m *Module) RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	adminGroup.GET("/fingerprints/associations", m.Handler.ListFingerprintAssociations)
}
