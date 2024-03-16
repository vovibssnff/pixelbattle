package websockets

import (
	"net/http"
	"pb_backend/models"
	"pb_backend/service"
	"time"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Client struct {
	conn          *websocket.Conn
	server        *WsServer
	send          chan *models.Pixel
	err           chan *string
	userid        int
	timer_service *redis.Client
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

func NewClient(conn *websocket.Conn, server *WsServer, id int, redisClient *redis.Client) *Client {
	return &Client{
		conn:          conn,
		server:        server,
		send:          make(chan *models.Pixel),
		err:           make(chan *string),
		userid:        id,
		timer_service: redisClient,
	}
}

func ServeWs(server *WsServer, redisClient *redis.Client, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	session, _ := server.store.Get(r, "user-session")

	if session.Values["Authenticated"] != "true" {
		logrus.Warn("Unauthorized attempt to reach /ws")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err != nil {
		logrus.Error(err)
		return
	}

	client := NewClient(conn, server, session.Values["ID"].(int), redisClient)

	go client.writePump()
	go client.readPump()

	server.register <- client
}

func (client *Client) readPump() {
	logrus.Info("ReadPump routine running")
	defer func() {
		logrus.Info("Read pump disconnected")
		client.disconnect()
	}()
	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error { client.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, msg, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Error(err)
			}
			break
		}

		var deserialized models.Pixel
		if err := deserialized.Deserialize(msg); err != nil {
			logrus.Error(err)
			continue
		}
		deserialized.Userid = client.userid

		//таймер чек по client.userid
		exists, err := service.CheckTime(client.timer_service, client.userid)
		if err != nil {
			logrus.Error(err)
		}

		// if doesn't exist
		if exists == 0 {
			err := service.SetTimer(client.timer_service, client.userid, 2)
			if err != nil {
				logrus.Error(err)
			}
			client.server.broadcast <- &deserialized
		}
	}
}

func (client *Client) writePump() {
	logrus.Info("WritePump routine running")
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		logrus.Info("Write pump disconnected")
		ticker.Stop()
		client.conn.Close()
	}()
	for {
		select {
		case pixel, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			serialized, err := pixel.Serialize()
			if err != nil {
				logrus.Error(err)
				return
			}
			w.Write(serialized)
			logrus.Info("WritePump sent: ", serialized)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *Client) disconnect() {
	client.server.unregister <- client
	close(client.send)
	client.conn.Close()
}
