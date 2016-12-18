package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/asim/mq/go/client"
	"github.com/gorilla/handlers"
	"github.com/gorilla/websocket"
)

type mq struct {
	client client.Client

	sync.RWMutex
	topics map[string][]chan []byte
}

var (
	address = flag.String("address", ":8081", "MQ server address")
	cert    = flag.String("cert_file", "", "TLS certificate file")
	key     = flag.String("key_file", "", "TLS key file")
	proxy   = flag.Bool("proxy", false, "Proxy for an MQ cluster")
	servers = flag.String("servers", "", "Comma separated MQ cluster list used by Proxy")

	defaultMQ *mq
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

func init() {
	flag.Parse()

	if *proxy && len(*servers) == 0 {
		log.Fatal("Proxy enabled without MQ server list")
	}

	defaultMQ = &mq{
		client: client.New(client.WithServers(strings.Split(*servers, ",")...)),
		topics: make(map[string][]chan []byte),
	}
}

func (m *mq) pub(topic string, payload []byte) error {
	if *proxy {
		return m.client.Publish(topic, payload)
	}

	m.RLock()
	subscribers, ok := m.topics[topic]
	m.RUnlock()
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

func (m *mq) sub(topic string) (<-chan []byte, error) {
	if *proxy {
		return m.client.Subscribe(topic)
	}

	ch := make(chan []byte, 100)
	m.Lock()
	m.topics[topic] = append(m.topics[topic], ch)
	m.Unlock()
	return ch, nil
}

func (m *mq) unsub(topic string, sub <-chan []byte) error {
	if *proxy {
		return m.client.Unsubscribe(sub)
	}

	m.RLock()
	subscribers, ok := m.topics[topic]
	m.RUnlock()

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

	m.Lock()
	m.topics[topic] = subs
	m.Unlock()

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
	var wr writer

	if u := r.Header.Get("Upgrade"); len(u) == 0 || u != "websocket" {
		wr = &httpWriter{w}
	} else {
		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
			return
		}
		wr = &wsWriter{conn}
	}

	topic := r.URL.Query().Get("topic")
	ch, err := defaultMQ.sub(topic)
	if err != nil {
		http.Error(w, "Could not retrieve events", http.StatusInternalServerError)
		return
	}
	defer defaultMQ.unsub(topic, ch)

	for {
		select {
		case e := <-ch:
			if err = wr.Write(e); err != nil {
				return
			}
		}
	}
}

func main() {
	// MQ Handlers
	http.HandleFunc("/pub", pub)
	http.HandleFunc("/sub", sub)

	// logging handler
	handler := handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)

	if len(*cert) > 0 && len(*key) > 0 {
		log.Println("TLS Enabled")
		log.Println("MQ listening on", *address)
		http.ListenAndServeTLS(*address, *cert, *key, handler)
		return
	}

	if *proxy {
		log.Println("Proxy enabled")
	}

	log.Println("MQ listening on", *address)
	http.ListenAndServe(*address, handler)
}
