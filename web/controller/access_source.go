package controller

import (
	"github.com/gin-gonic/gin"
	"x-ui/web/service"
)

type AccessSourceController struct {
	BaseController

	accessSourceService service.AccessSourceService
}

func NewAccessSourceController(g *gin.RouterGroup) *AccessSourceController {
	a := &AccessSourceController{}
	a.initRouter(g)
	return a
}

func (a *AccessSourceController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/api/access-source")
	g.Use(a.checkLogin)
	g.GET("/analytics", a.analytics)
}

func (a *AccessSourceController) analytics(c *gin.Context) {
	report, err := a.accessSourceService.Analyze()
	jsonMsgObj(c, "获取访问来源分析", report, err)
}
