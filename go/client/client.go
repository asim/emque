package client

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	mqServer = "127.0.0.1:8081"
)

func Publish(topic string, payload []byte) error {
	rsp, err := http.Post(fmt.Sprintf("http://%s/pub?topic=%s", mqServer, topic), "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return fmt.Errorf("Non 200 response %d", rsp.StatusCode)
	}
	return nil
}

func Subscribe(topic string) (<-chan []byte, error) {
	c, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/sub?topic=%s", mqServer, topic), make(http.Header))
	if err != nil {
		return nil, err
	}

	ch := make(chan []byte, 100)

	go func() {
		for {
			t, p, err := c.ReadMessage()
			if err != nil {
				log.Println("Failed to read message, closing channel", err)
				c.Close()
				close(ch)
				return
			}
			switch t {
			case websocket.CloseMessage:
				log.Println("Close message, closing channel")
				c.Close()
				close(ch)
				return
			default:
				ch <- p
			}
		}
	}()

	return ch, nil
}
