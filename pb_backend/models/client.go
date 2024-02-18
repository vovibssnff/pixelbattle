package models

import (
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type Client struct {
	conn   *websocket.Conn
	server *WsServer
	send   chan *Pixel
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// var (
// 	newline = []byte{'\n'}
// 	space   = []byte{' '}
// )

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 10000
)

func NewClient(conn *websocket.Conn, server *WsServer) *Client {
	return &Client{
		conn:   conn,
		server: server,
		send:   make(chan *Pixel),
	}
}

func ServeWs(server *WsServer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Error(err)
		return
	}

	client := NewClient(conn, server)

	go client.writePump()
	go client.readPump()

	server.register <- client

}

func (client *Client) readPump() {
	logrus.Info("ReadPump routine running")
	defer func() {
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
		logrus.Info("ReadPump received: ", msg)
		client.server.broadcast <- msg
	}
}

func (client *Client) writePump() {
	logrus.Info("WritePump routine running")
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()
	for {
		select {
		case pixel, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The WsServer closed the channel.
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
			// n := len(client.send)
			// for i := 0; i < n; i++ {
			// 	w.Write(newline)
			// 	w.Write(<-client.send)
			// }

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
