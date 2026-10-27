// Package api wires the HTTP surface: /api/* JSON + WebSocket, everything
// else falls through to the embedded single-page app.
package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skinnyrad/Lora-Scanner/internal/frames"
	"github.com/skinnyrad/Lora-Scanner/internal/gateway"
	"github.com/skinnyrad/Lora-Scanner/internal/hub"
	"github.com/skinnyrad/Lora-Scanner/internal/serial"
)

type Server struct {
	Serial  *serial.Manager
	Hub     *hub.Hub
	Frames  *frames.Store
	Config  *gateway.Config
	Scraper *gateway.Scraper
	UI      fs.FS // contents of dist/, rooted at index.html
}

func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	api := r.Group("/api")
	api.GET("/ports", s.ports)
	api.GET("/transceivers", s.transceiversList)
	api.POST("/transceivers", s.transceiverAdd)
	api.DELETE("/transceivers/:id", s.transceiverRemove)
	api.POST("/transceivers/:id/transmit", s.transceiverTransmit)
	api.GET("/transceivers/:id/beacon", s.beaconGet)
	api.POST("/transceivers/:id/beacon", s.beaconStart)
	api.DELETE("/transceivers/:id/beacon", s.beaconStop)
	api.GET("/gateways", s.gatewaysGet)
	api.POST("/gateways", s.gatewayAdd)
	api.DELETE("/gateways/:ip", s.gatewayRemove)
	api.GET("/gateways/status", s.gatewaysStatus)
	api.GET("/gateways/poll", s.pollGet)
	api.PUT("/gateways/poll", s.pollPut)
	api.GET("/frames", s.framesGet)
	api.DELETE("/frames", s.framesClear)
	api.GET("/frames.csv", s.framesCSV)
	api.GET("/devices", s.devicesGet)
	api.GET("/ws", s.ws)

	r.NoRoute(s.noRoute)
	return r
}

func (s *Server) noRoute(c *gin.Context) {
	p := c.Request.URL.Path
	if p == "/api" || strings.HasPrefix(p, "/api/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	name := strings.TrimPrefix(path.Clean(p), "/")
	if name != "" {
		if info, err := fs.Stat(s.UI, name); err == nil && !info.IsDir() {
			c.FileFromFS(name, http.FS(s.UI))
			return
		}
	}
	index, err := fs.ReadFile(s.UI, "index.html")
	if err != nil {
		c.String(http.StatusServiceUnavailable, "frontend not built: run `make web`")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", index)
}
