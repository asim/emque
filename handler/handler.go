package handler

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/asim/mq/broker"
	"github.com/gorilla/websocket"
)

func Pub(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")

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

func Sub(w http.ResponseWriter, r *http.Request) {
	var wr writer

	if u := r.Header.Get("Upgrade"); len(u) == 0 || u != "websocket" {
		wr = &httpWriter{w}
	} else {
		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not open websocket connection: %v", err), http.StatusBadRequest)
			return
		}
		wr = &wsWriter{conn}
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
