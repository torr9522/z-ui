package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestAccessSourceAnalyticsRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	engine.Use(func(c *gin.Context) {
		c.Set("base_path", "/")
	})
	NewAccessSourceController(engine.Group("/"))

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, "/api/access-source/analytics", nil)
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "登录时效已过") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}
