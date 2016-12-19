package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	mqclient "github.com/asim/mq/go/client"
	"github.com/gorilla/handlers"
	"github.com/gorilla/websocket"
)

type mq struct {
	client mqclient.Client

	sync.RWMutex
	topics map[string][]chan []byte
}

type writer interface {
	Write(b []byte) error
}

type httpWriter struct {
	w http.ResponseWriter
}

type wsWriter struct {
	conn *websocket.Conn
}

var (
	address = flag.String("address", ":8081", "MQ server address")
	cert    = flag.String("cert_file", "", "TLS certificate file")
	key     = flag.String("key_file", "", "TLS key file")

	// server persist to file
	persist = flag.Bool("persist", false, "Persist messages to [topic].mq file per topic")

	// proxy flags
	proxy   = flag.Bool("proxy", false, "Proxy for an MQ cluster")
	retries = flag.Int("retries", 1, "Number of retries for publish or subscribe")
	servers = flag.String("servers", "", "Comma separated MQ cluster list used by Proxy")

	// client flags
	client    = flag.Bool("client", false, "Run the MQ client")
	publish   = flag.Bool("publish", false, "Publish via the MQ client")
	subscribe = flag.Bool("subscribe", false, "Subscribe via the MQ client")
	topic     = flag.String("topic", "", "Topic for client to publish or subscribe to")

	defaultMQ *mq
)

func init() {
	flag.Parse()

	if *proxy && *client {
		log.Fatal("Client and proxy flags cannot be specified together")
	}

	if *proxy && len(*servers) == 0 {
		log.Fatal("Proxy enabled without MQ server list")
	}

	if *client && len(*topic) == 0 {
		log.Fatal("Topic not specified")
	}

	if *client && !*publish && !*subscribe {
		log.Fatal("Specify whether to publish or subscribe")
	}

	if *client && len(*servers) == 0 {
		log.Fatal("Client specified without MQ server list")
	}

	defaultMQ = &mq{
		client: mqclient.New(
			mqclient.WithServers(strings.Split(*servers, ",")...),
			mqclient.WithRetries(*retries),
		),
		topics: make(map[string][]chan []byte),
	}
}

func (m *mq) persist(topic string) error {
	ch, err := m.sub(topic)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(topic+".mq", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		return err
	}

	go func() {
		for b := range ch {
			f.Write(b)
			f.Write([]byte{'\n'})
		}
	}()

	return nil
}

func (m *mq) pub(topic string, payload []byte) error {
	if *proxy {
		return m.client.Publish(topic, payload)
	}

	m.RLock()
	subscribers, ok := m.topics[topic]
	m.RUnlock()
	if !ok {
		// persist?
		if !*persist {
			return nil
		}
		if err := m.persist(topic); err != nil {
			return err
		}
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

func cli() {
	// process publish
	if *publish {
		// scan till EOF
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			defaultMQ.client.Publish(*topic, scanner.Bytes())
		}
	}

	if !*subscribe {
		return
	}

	// process subscribe
	ch, err := defaultMQ.client.Subscribe(*topic)
	if err != nil {
		fmt.Println(err)
		return
	}
	for e := range ch {
		fmt.Println(string(e))
	}
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
			http.Error(w, fmt.Sprintf("Could not open websocket connection: %v", err), http.StatusBadRequest)
			return
		}
		wr = &wsWriter{conn}
	}

	topic := r.URL.Query().Get("topic")
	ch, err := defaultMQ.sub(topic)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not retrieve events: %v", err), http.StatusInternalServerError)
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
	// handle client
	if *client {
		cli()
		return
	}

	// MQ Handlers
	http.HandleFunc("/pub", pub)
	http.HandleFunc("/sub", sub)

	// logging handler
	handler := handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)

	if len(*cert) > 0 && len(*key) > 0 {
		log.Println("TLS Enabled")
		log.Println("MQ listening on", *address)
		err := http.ListenAndServeTLS(*address, *cert, *key, handler)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if *proxy {
		log.Println("Proxy enabled")
	}

	log.Println("MQ listening on", *address)
	err := http.ListenAndServe(*address, handler)
	if err != nil {
		log.Fatal(err)
	}
}
