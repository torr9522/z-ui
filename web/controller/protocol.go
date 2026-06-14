package controller

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"os/exec"
	"strings"
	"time"
	"x-ui/protocol"
	"x-ui/xray"
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
	g.POST("/reality/keypair", a.checkCSRF, a.realityKeyPair)
}

func (a *ProtocolController) schema(c *gin.Context) {
	module, err := protocol.DefaultRegistry().Get(c.Param("protocol"))
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, module.FormSchema(), nil)
}

func (a *ProtocolController) realityKeyPair(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, xray.GetBinaryPath(), "x25519").CombinedOutput()
	if err != nil {
		jsonObj(c, nil, err)
		return
	}

	privateKey, publicKey := parseRealityKeyPair(string(output))
	if privateKey == "" || publicKey == "" {
		jsonObj(c, nil, errors.New("unable to parse reality key pair"))
		return
	}

	jsonObj(c, gin.H{
		"privateKey": privateKey,
		"publicKey":  publicKey,
	}, nil)
}

func parseRealityKeyPair(output string) (string, string) {
	var privateKey string
	var publicKey string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "PrivateKey:"):
			privateKey = strings.TrimSpace(strings.TrimPrefix(line, "PrivateKey:"))
		case strings.HasPrefix(line, "Password (PublicKey):"):
			publicKey = strings.TrimSpace(strings.TrimPrefix(line, "Password (PublicKey):"))
		}
	}
	return privateKey, publicKey
}
