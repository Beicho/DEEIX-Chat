package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/notification"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	notificationrepo "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/postgres/notification"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNotificationRoutesSupportPatchMarkRead(t *testing.T) {
	router, service := newNotificationTestRouter(t)
	item, err := service.Create(t.Context(), 7, appnotification.CreateInput{Title: "A", Body: "B", Link: "/settings"})
	if err != nil {
		t.Fatalf("create notification: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/notifications/"+item.ID+"/read", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH mark read status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestNotificationListResponseKeepsProducerFieldsInternal(t *testing.T) {
	router, service := newNotificationTestRouter(t)
	_, err := service.Create(t.Context(), 7, appnotification.CreateInput{
		Title:    "A",
		Body:     "B",
		Link:     "/settings",
		Source:   "billing",
		SourceID: "order-1",
		Metadata: map[string]any{"internal": "value"},
	})
	if err != nil {
		t.Fatalf("create notification: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("response data missing: %s", rec.Body.String())
	}
	results, ok := data["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("response results = %#v", data["results"])
	}
	result, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("response result = %#v", results[0])
	}
	if result["type"] != "system" {
		t.Fatalf("type = %q, body = %s", result["type"], rec.Body.String())
	}
	body := rec.Body.String()
	for _, field := range []string{`"metadata"`, `"actionURL"`, `"source"`, `"sourceId"`} {
		if strings.Contains(body, field) {
			t.Fatalf("response exposes internal field %s: %s", field, body)
		}
	}
}

func newNotificationTestRouter(t *testing.T) (*gin.Engine, *appnotification.Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:notification_http?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("migrate notification table: %v", err)
	}

	service := appnotification.NewService(notificationrepo.NewRepo(db), nil)
	module := NewModule(NewHandler(service))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, uint(7))
		c.Next()
	})
	module.RegisterRoutes(router.Group(""))
	return router, service
}
