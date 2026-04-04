package websockets

import (
	"context"
	"net/http"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Client struct {
	conn         *websocket.Conn
	server       *WsServer
	send         chan *domain.Pixel
	userid       int
	faculty      string
	isAdm        bool
	timerService domain.TimerService
	userService  domain.UserService
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
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
	userid int,
	faculty string,
	isAdm bool,
	timerService domain.TimerService,
	userService domain.UserService,
) *Client {
	return &Client{
		conn:         conn,
		server:       server,
		send:         make(chan *domain.Pixel),
		userid:       userid,
		faculty:      faculty,
		isAdm:        isAdm,
		timerService: timerService,
		userService:  userService,
	}
}

func ServeWs(server *WsServer, w http.ResponseWriter, r *http.Request) {
	var userid int
	var faculty string
	var isAdm bool

	if server.allowAnonymousWS {
		// Load-test mode: no session. k6 passes uid & faculty query params (per-VU identity, timer like real users).
		q := r.URL.Query()
		uidStr := q.Get("uid")
		if uidStr == "" {
			http.Error(w, "missing query: uid (required for anonymous benchmark WebSocket)", http.StatusBadRequest)
			return
		}
		parsed, err := strconv.Atoi(uidStr)
		if err != nil || parsed <= 0 {
			http.Error(w, "invalid uid", http.StatusBadRequest)
			return
		}
		userid = parsed
		faculty = q.Get("faculty")
		if faculty == "" {
			faculty = "KTU"
		}
		switch faculty {
		case "KTU", "TINT", "FTMF", "FTMI", "NOZH":
		default:
			http.Error(w, "invalid faculty (use KTU|TINT|FTMF|FTMI|NOZH)", http.StatusBadRequest)
			return
		}
		isAdm = server.userService.IsAdmin(userid)
		if server.userService.IsUserBanned(r.Context(), userid) {
			logrus.Info("Anonymous WS rejected: banned uid ", userid)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		logrus.Debugf("WebSocket anonymous benchmark uid=%d faculty=%s isAdm=%v", userid, faculty, isAdm)
	} else {
		session, err := server.sessionService.GetSession(r)
		if err != nil {
			logrus.Error("Failed to get session: ", err)
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
		isAdm = server.userService.IsAdmin(userid)

		if server.userService.IsUserBanned(r.Context(), userid) {
			logrus.Info("Request from banned user: ", userid)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error(err)
		return
	}

	client := NewClient(
		conn,
		server,
		userid,
		faculty,
		isAdm,
		server.timerService,
		server.userService,
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
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Error(err)
			}
			break
		}

		var pixel domain.Pixel
		if err = utils.DeserializePixel(msg, &pixel); err != nil {
			logrus.Error(err)
			continue
		}
		pixel.Userid = c.userid
		pixel.Faculty = c.faculty

		if c.isAdm {
			c.server.broadcast <- &pixel
		} else if c.userService.IsUserBanned(ctx, c.userid) {
			logrus.Info("Request from banned user: ", c.userid)
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

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case pixel, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			serialized, err := utils.SerializePixel(pixel)
			if err != nil {
				logrus.Error(err)
				return
			}
			w.Write(serialized)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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