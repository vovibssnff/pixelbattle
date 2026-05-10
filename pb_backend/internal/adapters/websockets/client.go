package websockets

import (
	"context"
	"net/http"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/core/service"
	"pb_backend/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Client struct {
	conn           *websocket.Conn
	server         *WsServer
	send           chan *domain.Pixel
	initialReplay  []*domain.Pixel
	wireV2         bool
	userid         string
	faculty        string
	isAdm          bool
	timerService   domain.TimerService
	userService    domain.UserService
	canvasWidth    uint
	canvasHeight   uint
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:      4096,
	WriteBufferSize:     4096,
	EnableCompression:   true,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
	// Prefer v2 (binary pixels); client may omit header → no subprotocol → JSON v1.
	Subprotocols: []string{SubprotocolV2, SubprotocolV1},
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 10000
)

func NewClient(
	conn *websocket.Conn,
	server *WsServer,
	userid string,
	faculty string,
	isAdm bool,
	timerService domain.TimerService,
	userService domain.UserService,
	canvasWidth, canvasHeight uint,
	initialReplay []*domain.Pixel,
	wireV2 bool,
) *Client {
	return &Client{
		conn:          conn,
		server:        server,
		send:          make(chan *domain.Pixel, 256),
		initialReplay: initialReplay,
		wireV2:        wireV2,
		userid:        userid,
		faculty:       faculty,
		isAdm:         isAdm,
		timerService:  timerService,
		userService:   userService,
		canvasWidth:   canvasWidth,
		canvasHeight:  canvasHeight,
	}
}

func validPixel(p *domain.Pixel, canvasW, canvasH uint) bool {
	if canvasW == 0 || canvasH == 0 {
		return false
	}
	if p.X >= canvasW || p.Y >= canvasH {
		return false
	}
	if len(p.Color) != 3 {
		return false
	}
	for _, c := range p.Color {
		if c > 255 {
			return false
		}
	}
	return true
}

// benchmarkUIDToCanonical maps k6 "uid" query to a canonical user id (numeric -> vk_*).
func parseReplayAfterMS(r *http.Request) int64 {
	q := strings.TrimSpace(r.URL.Query().Get("replay_after_ms"))
	if q == "" {
		return 0
	}
	v, err := strconv.ParseInt(q, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func benchmarkUIDToCanonical(uidStr string) (string, bool) {
	uidStr = strings.TrimSpace(uidStr)
	if uidStr == "" {
		return "", false
	}
	if n, err := strconv.Atoi(uidStr); err == nil && n > 0 {
		return domain.VKUserID(n), true
	}
	u := utils.NormalizeUsername(uidStr)
	if u == "" {
		return "", false
	}
	return u, true
}

func ServeWs(server *WsServer, w http.ResponseWriter, r *http.Request) {
	var userid string
	var faculty string
	var isAdm bool

	if server.allowAnonymousWS {
		q := r.URL.Query()
		uidStr := q.Get("uid")
		var ok bool
		userid, ok = benchmarkUIDToCanonical(uidStr)
		if !ok {
			http.Error(w, "missing or invalid query: uid", http.StatusBadRequest)
			return
		}
		faculty = q.Get("faculty")
		if faculty == "" {
			faculty = "KTU"
		}
		if !utils.IsValidFaculty(faculty) {
			http.Error(w, "invalid faculty (use KTU|TINT|FTMF|FTMI|NOZH)", http.StatusBadRequest)
			return
		}
		isAdm = server.userService.IsEffectiveAdmin(r.Context(), userid)
		if server.userService.IsUserBanned(r.Context(), userid) {
			logrus.Info("Anonymous WS rejected: banned ", userid)
			service.IncrementBannedRejected()
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		logrus.Debugf("WebSocket anonymous benchmark uid=%s faculty=%s isAdm=%v", userid, faculty, isAdm)
	} else {
		session, err := server.sessionService.GetSession(r)
		if err != nil {
			logrus.Error("Failed to get session: ", err)
			service.IncrementSessionErrors()
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !server.sessionService.IsAuthenticated(session) {
			logrus.Warn("Unauthorized attempt to reach /ws")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userid = server.sessionService.GetUserID(session)
		faculty = server.sessionService.GetFaculty(session)
		isAdm = server.userService.IsEffectiveAdmin(r.Context(), userid)

		if server.userService.IsUserBanned(r.Context(), userid) {
			logrus.Info("Request from banned user: ", userid)
			service.IncrementBannedRejected()
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	if server.limiter != nil && !server.limiter.AllowWSConn(clientIP(r)) {
		service.IncrementRejected("rate_limit_ws_ip")
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error(err)
		return
	}

	var initialReplay []*domain.Pixel
	if ra := parseReplayAfterMS(r); ra > 0 && server.replay != nil {
		initialReplay = server.replay.Since(ra)
	}

	wireV2 := conn.Subprotocol() == SubprotocolV2

	client := NewClient(
		conn,
		server,
		userid,
		faculty,
		isAdm,
		server.timerService,
		server.userService,
		server.canvasWidth,
		server.canvasHeight,
		initialReplay,
		wireV2,
	)

	go client.writePump()
	go client.readPump(r.Context())

	server.register <- client
}

func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.disconnect()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		mt, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Error(err)
				service.IncrementWSError("unexpected_close")
			}
			service.IncrementWSError("read")
			break
		}
		service.IncrementWebSocketMessagesReceived("pixel")
		serverRecvMs := time.Now().UnixMilli()

		var pixel domain.Pixel
		switch mt {
		case websocket.BinaryMessage:
			pixel, err = decodeClientPixelV2(msg)
			if err != nil {
				logrus.Debug(err)
				service.IncrementWSError("deserialize_binary")
				continue
			}
		case websocket.TextMessage:
			if err = utils.DeserializePixel(msg, &pixel); err != nil {
				logrus.Error(err)
				service.IncrementWSError("deserialize")
				continue
			}
		default:
			continue
		}
		if !validPixel(&pixel, c.canvasWidth, c.canvasHeight) {
			logrus.Warn("invalid pixel rejected")
			service.IncrementWSError("invalid_pixel")
			continue
		}
		if service.IsCanvasFrozen() {
			service.IncrementWSError("frozen")
			continue
		}
		pixel.ServerRecvMs = serverRecvMs
		if pixel.ClientSentMs > 0 && serverRecvMs > pixel.ClientSentMs {
			service.ObserveE2EPixelLatency("ws_client_to_server", time.Duration(serverRecvMs-pixel.ClientSentMs)*time.Millisecond)
		}
		pixel.Userid = c.userid
		pixel.Faculty = c.faculty

		if !c.isAdm && c.server.limiter != nil && !c.server.limiter.AllowPlacement(c.userid) {
			service.IncrementRejected("rate_limit_pixel")
			continue
		}

		if c.isAdm {
			c.server.broadcast <- &pixel
		} else if c.userService.IsUserBanned(ctx, c.userid) {
			logrus.Info("Request from banned user: ", c.userid)
			service.IncrementBannedRejected()
			return
		} else {
			exists, err := c.timerService.CheckTime(ctx, c.userid)
			if err != nil {
				logrus.Error(err)
			}

			if exists == 0 {
				err := c.timerService.SetTimer(ctx, c.userid)
				if err != nil {
					logrus.Error(err)
				}
				c.server.broadcast <- &pixel
			}
		}
	}
}

func (c *Client) writeBinary(b []byte) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		service.IncrementWSError("write")
		return err
	}
	return nil
}

func (c *Client) writePixelOut(pixel *domain.Pixel) error {
	if c.wireV2 {
		if b := encodePixelV2(pixel); b != nil {
			return c.writeBinary(b)
		}
	}
	return c.writePixelJSON(pixel)
}

func (c *Client) writePixelJSON(pixel *domain.Pixel) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		service.IncrementWSError("next_writer")
		return err
	}
	serialized, err := utils.SerializePixel(pixel)
	if err != nil {
		logrus.Error(err)
		service.IncrementWSError("serialize")
		return err
	}
	if _, err := w.Write(serialized); err != nil {
		service.IncrementWSError("write")
		return err
	}
	if err := w.Close(); err != nil {
		service.IncrementWSError("writer_close")
		return err
	}
	return nil
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for _, p := range c.initialReplay {
		if err := c.writePixelOut(p); err != nil {
			return
		}
	}
	c.initialReplay = nil

	for {
		select {
		case pixel, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.writePixelOut(pixel); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				service.IncrementWSError("ping")
				return
			}
		}
	}
}

func (c *Client) disconnect() {
	c.server.unregister <- c
	close(c.send)
	c.conn.Close()
}
