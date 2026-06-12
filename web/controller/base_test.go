package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"x-ui/web/session"
)

func TestCheckCSRFMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	engine.Use(sessions.Sessions("session", store))
	base := &BaseController{}

	engine.GET("/seed", func(c *gin.Context) {
		token, err := session.EnsureCSRFToken(c)
		if err != nil {
			t.Fatalf("ensure csrf token: %v", err)
		}
		c.Header("X-Test-CSRF-Token", token)
		c.Status(http.StatusNoContent)
	})
	engine.POST("/write", base.checkCSRF, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, "/seed", nil)
	engine.ServeHTTP(recorder, request)

	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
	token := recorder.Header().Get("X-Test-CSRF-Token")
	if token == "" {
		t.Fatal("expected csrf token header")
	}

	invalid := httptest.NewRecorder()
	request, _ = http.NewRequest(http.MethodPost, "/write", nil)
	request.AddCookie(cookies[0])
	engine.ServeHTTP(invalid, request)
	if invalid.Code != http.StatusOK {
		t.Fatalf("unexpected invalid csrf status: %d", invalid.Code)
	}

	valid := httptest.NewRecorder()
	request, _ = http.NewRequest(http.MethodPost, "/write", nil)
	request.AddCookie(cookies[0])
	request.Header.Set("X-CSRF-Token", token)
	engine.ServeHTTP(valid, request)
	if valid.Code != http.StatusNoContent {
		t.Fatalf("unexpected valid csrf status: %d", valid.Code)
	}
}
