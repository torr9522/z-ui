package controller

import (
	"github.com/gin-gonic/gin"
	"x-ui/protocol"
)

type ProtocolController struct {
	BaseController
}

func NewProtocolController(g *gin.RouterGroup) *ProtocolController {
	a := &ProtocolController{}
	a.initRouter(g)
	return a
}

func (a *ProtocolController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/api/protocol")
	g.Use(a.checkLogin)

	g.GET("/schema/:protocol", a.schema)
}

func (a *ProtocolController) schema(c *gin.Context) {
	module, err := protocol.DefaultRegistry().Get(c.Param("protocol"))
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, module.FormSchema(), nil)
}
