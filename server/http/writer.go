package http

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type writer interface {
	Write(b []byte) error
}

type httpWriter struct {
	w http.ResponseWriter
}

type wsWriter struct {
	conn *websocket.Conn
}

func (w *httpWriter) Write(b []byte) error {
	if _, err := w.w.Write(b); err != nil {
		return err
	}
	if f, ok := w.w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

func (w *wsWriter) Write(b []byte) error {
	return w.conn.WriteMessage(websocket.BinaryMessage, b)
}
