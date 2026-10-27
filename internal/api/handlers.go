package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/skinnyrad/Lora-Scanner/internal/beacon"
	"github.com/skinnyrad/Lora-Scanner/internal/frames"
	"github.com/skinnyrad/Lora-Scanner/internal/gateway"
	"github.com/skinnyrad/Lora-Scanner/internal/serial"
)

func fail(c *gin.Context, code int, err error) {
	c.JSON(code, gin.H{"error": err.Error()})
}

// serialErr maps manager errors to HTTP statuses.
func serialErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, serial.ErrNotFound):
		fail(c, http.StatusNotFound, err)
	case errors.Is(err, serial.ErrPortInUse):
		fail(c, http.StatusConflict, err)
	default:
		fail(c, http.StatusInternalServerError, err)
	}
}

func idParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		fail(c, http.StatusNotFound, serial.ErrNotFound)
		return 0, false
	}
	return id, true
}

func (s *Server) ports(c *gin.Context) {
	ports, err := serial.ListPorts()
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	if ports == nil {
		ports = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"ports": ports})
}

func (s *Server) transceiversList(c *gin.Context) { c.JSON(http.StatusOK, s.Serial.List()) }

func (s *Server) transceiverAdd(c *gin.Context) {
	var body struct {
		Port string `json:"port"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	t, err := s.Serial.Add(body.Port)
	if err != nil {
		serialErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (s *Server) transceiverRemove(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := s.Serial.Remove(id); err != nil {
		serialErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) transceiverTransmit(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := s.Serial.Transmit(id, body.Text); err != nil {
		serialErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": "ok"})
}

func (s *Server) beaconGet(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	st, err := s.Serial.BeaconState(id)
	if err != nil {
		serialErr(c, err)
		return
	}
	c.JSON(http.StatusOK, st)
}

func (s *Server) beaconStart(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body struct {
		Text       string `json:"text"`
		IntervalMs int64  `json:"intervalMs"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	d := time.Duration(body.IntervalMs) * time.Millisecond
	if d < beacon.MinInterval {
		fail(c, http.StatusBadRequest, errors.New("intervalMs must be at least 1000"))
		return
	}
	if err := s.Serial.StartBeacon(id, body.Text, d); err != nil {
		serialErr(c, err)
		return
	}
	st, _ := s.Serial.BeaconState(id)
	c.JSON(http.StatusOK, st)
}

func (s *Server) beaconStop(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := s.Serial.StopBeacon(id); err != nil {
		serialErr(c, err)
		return
	}
	st, _ := s.Serial.BeaconState(id)
	c.JSON(http.StatusOK, st)
}

func (s *Server) gatewaysGet(c *gin.Context) { c.JSON(http.StatusOK, s.Config.List()) }

func (s *Server) gatewayAdd(c *gin.Context) {
	var body struct {
		IP string `json:"ip"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	ip, err := s.Config.Add(body.IP)
	switch {
	case errors.Is(err, gateway.ErrDuplicate):
		fail(c, http.StatusConflict, err)
	case err != nil:
		fail(c, http.StatusBadRequest, err)
	default:
		c.JSON(http.StatusCreated, gin.H{"ip": ip})
	}
}

func (s *Server) gatewayRemove(c *gin.Context) {
	if err := s.Config.Remove(c.Param("ip")); err != nil {
		if errors.Is(err, gateway.ErrNotFound) {
			fail(c, http.StatusNotFound, err)
		} else {
			fail(c, http.StatusInternalServerError, err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

type pollBody struct {
	Seconds int `json:"seconds"`
}

func (s *Server) pollGet(c *gin.Context) {
	c.JSON(http.StatusOK, pollBody{Seconds: int(s.Config.PollInterval() / time.Second)})
}

func (s *Server) pollPut(c *gin.Context) {
	var body pollBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	switch err := s.Config.SetPollInterval(time.Duration(body.Seconds) * time.Second); {
	case errors.Is(err, gateway.ErrBadInterval):
		fail(c, http.StatusBadRequest, err)
	case err != nil:
		fail(c, http.StatusInternalServerError, err)
	default:
		s.pollGet(c)
	}
}

func (s *Server) gatewaysStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.Scraper.Statuses())
}

func (s *Server) framesGet(c *gin.Context) {
	after, err := afterParam(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, s.Frames.Since(after))
}

func afterParam(c *gin.Context) (uint64, error) {
	v := c.Query("after")
	if v == "" {
		return 0, nil
	}
	return strconv.ParseUint(v, 10, 64)
}

func (s *Server) framesClear(c *gin.Context) {
	s.Frames.Clear()
	c.Status(http.StatusNoContent)
}

func (s *Server) framesCSV(c *gin.Context) {
	c.Header("Content-Disposition", `attachment; filename="frames.csv"`)
	c.Header("Content-Type", "text/csv")
	c.Status(http.StatusOK)
	_ = frames.WriteCSV(c.Writer, s.Frames.Since(0))
}

func (s *Server) devicesGet(c *gin.Context) { c.JSON(http.StatusOK, s.Frames.Devices()) }

var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 4096}

// ws streams the transceiver list, then live frame/device/transceivers messages. Clients send nothing;
// they seed their state from GET /api/frames and /api/devices.
func (s *Server) ws(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Subscribe first so nothing is missed between snapshot and stream.
	ch, cancel := s.Hub.Subscribe()
	defer cancel()

	if conn.WriteJSON(serial.ListMsg{Type: "transceivers", Transceivers: s.Serial.List()}) != nil {
		return
	}
	if conn.WriteJSON(s.Scraper.StatusMsg()) != nil {
		return
	}

	closed := make(chan struct{})
	go func() { // detect client close
		defer close(closed)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case msg, ok := <-ch:
			if !ok || conn.WriteJSON(msg) != nil {
				return
			}
		case <-closed:
			return
		}
	}
}
