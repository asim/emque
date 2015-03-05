package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type mq struct {
	mtx    sync.RWMutex
	topics map[string][]chan []byte
}

var (
	defaultMQ = &mq{
		topics: make(map[string][]chan []byte),
	}
)

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func (m *mq) pub(topic string, payload []byte) error {
	m.mtx.RLock()
	subscribers, ok := m.topics[topic]
	m.mtx.RUnlock()
	if !ok {
		return nil
	}

	go func() {
		for _, subscriber := range subscribers {
			select {
			case subscriber <- payload:
			default:
			}
		}
	}()

	return nil
}

func (m *mq) sub(topic string) (<-chan []byte, error) {
	ch := make(chan []byte, 100)
	m.mtx.Lock()
	m.topics[topic] = append(m.topics[topic], ch)
	m.mtx.Unlock()
	return ch, nil
}

func (m *mq) unsub(topic string, sub <-chan []byte) error {
	m.mtx.RLock()
	subscribers, ok := m.topics[topic]
	m.mtx.RUnlock()

	if !ok {
		return nil
	}

	var subs []chan []byte
	for _, subscriber := range subscribers {
		if subscriber == sub {
			continue
		}
		subs = append(subs, subscriber)
	}

	m.mtx.Lock()
	m.topics[topic] = subs
	m.mtx.Unlock()

	return nil
}

func pub(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Pub error", http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	err = defaultMQ.pub(topic, b)
	if err != nil {
		http.Error(w, "Pub error", http.StatusInternalServerError)
		return
	}
}

func sub(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		log.Println("Failed to open websocket connection")
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}

	topic := r.URL.Query().Get("topic")
	ch, err := defaultMQ.sub(topic)
	if err != nil {
		log.Printf("Failed to retrieve event for %s topic", topic)
		http.Error(w, "Could not retrieve events", http.StatusInternalServerError)
		return
	}
	defer defaultMQ.unsub(topic, ch)

	for {
		select {
		case e := <-ch:
			if err = conn.WriteMessage(websocket.BinaryMessage, e); err != nil {
				log.Printf("error sending event: %v", err.Error())
				return
			}
		}
	}
}

func main() {
	http.HandleFunc("/pub", pub)
	http.HandleFunc("/sub", sub)

	log.Println("MQ listening on :8081")
	http.ListenAndServe(":8081", Log(http.DefaultServeMux))
}
