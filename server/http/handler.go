package http

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/asim/emque/broker"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func pub(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")

	if websocket.IsWebSocketUpgrade(r) {
		conn, err := upgrader.Upgrade(w, r, w.Header())
		if err != nil {
			return
		}
		for {
			messageType, b, err := conn.ReadMessage()
			if messageType == -1 {
				return
			}
			if err != nil {
				continue
			}
			broker.Publish(topic, b)
		}
	} else {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Pub error", http.StatusInternalServerError)
			return
		}
		r.Body.Close()
		if err := broker.Publish(topic, b); err != nil {
			http.Error(w, "Pub error", http.StatusInternalServerError)
		}
	}
}

func sub(w http.ResponseWriter, r *http.Request) {
	var wr writer

	if websocket.IsWebSocketUpgrade(r) {
		conn, err := upgrader.Upgrade(w, r, w.Header())
		if err != nil {
			return
		}
		// Drain the websocket so that we handle pings and connection close
		go func(c *websocket.Conn) {
			for {
				if _, _, err := c.NextReader(); err != nil {
					c.Close()
					break
				}
			}
		}(conn)
		wr = &wsWriter{conn}
	} else {
		wr = &httpWriter{w}
	}

	topic := r.URL.Query().Get("topic")

	ch, err := broker.Subscribe(topic)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not retrieve events: %v", err), http.StatusInternalServerError)
		return
	}
	defer broker.Unsubscribe(topic, ch)

	for {
		select {
		case e := <-ch:
			if err = wr.Write(e); err != nil {
				return
			}
		}
	}
}
