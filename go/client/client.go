package client

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	// The default client
	Default = newClient()
	// The default server list
	Servers = []string{"http://127.0.0.1:8081"}
)

// internal client
type client struct {
	options Options

	sync.RWMutex
	subscribers map[<-chan []byte]*subscriber
}

// internal subscriber
type subscriber struct {
	wg    sync.WaitGroup
	ch    chan<- []byte
	exit  chan bool
	topic string
}

// Client is the interface provided by this package
type Client interface {
	Publish(topic string, payload []byte) error
	Subscribe(topic string) (<-chan []byte, error)
	Unsubscribe(<-chan []byte) error
}

func newClient(opts ...Option) *client {
	options := Options{
		Servers: Servers,
	}

	for _, o := range opts {
		o(&options)
	}

	var servers []string

	for _, addr := range options.Servers {
		if !strings.HasPrefix(addr, "http") {
			addr = fmt.Sprintf("http://%s", addr)
		}
		servers = append(servers, addr)
	}

	// set servers
	WithServers(servers...)(&options)

	return &client{
		options:     options,
		subscribers: make(map[<-chan []byte]*subscriber),
	}
}

func publish(addr, topic string, payload []byte) error {
	url := fmt.Sprintf("%s/pub?topic=%s", addr, topic)
	rsp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return fmt.Errorf("Non 200 response %d", rsp.StatusCode)
	}
	return nil
}

func subscribe(addr string, s *subscriber) error {
	if strings.HasPrefix(addr, "http") {
		addr = strings.TrimPrefix(addr, "http")
		addr = "ws" + addr
	}

	url := fmt.Sprintf("%s/sub?topic=%s", addr, s.topic)
	c, _, err := websocket.DefaultDialer.Dial(url, make(http.Header))
	if err != nil {
		return err
	}

	go func() {
		defer s.wg.Done()

		for {
			t, p, err := c.ReadMessage()
			if err != nil || t == websocket.CloseMessage {
				c.Close()
				return
			}

			select {
			case <-s.exit:
				c.Close()
				return
			default:
				s.ch <- p
			}
		}
	}()

	return nil
}

func (c *client) Publish(topic string, payload []byte) error {
	var grr error
	for _, addr := range c.options.Servers {
		if err := publish(addr, topic, payload); err != nil {
			grr = err
		}
	}
	return grr
}

func (c *client) Subscribe(topic string) (<-chan []byte, error) {
	ch := make(chan []byte, len(c.options.Servers)*100)

	s := &subscriber{
		ch:    ch,
		exit:  make(chan bool),
		topic: topic,
	}

	var grr error
	for _, addr := range c.options.Servers {
		if err := subscribe(addr, s); err != nil {
			grr = err
			continue
		}
		s.wg.Add(1)
	}

	return ch, grr
}

func (c *client) Unsubscribe(ch <-chan []byte) error {
	c.Lock()
	defer c.Unlock()
	if sub, ok := c.subscribers[ch]; ok {
		return sub.Close()
	}
	return nil
}

func (s *subscriber) Close() error {
	select {
	case <-s.exit:
	default:
		close(s.exit)
		s.wg.Wait()
	}
	return nil
}

// Publish via the default Client
func Publish(topic string, payload []byte) error {
	return Default.Publish(topic, payload)
}

// Subscribe via the default Client
func Subscribe(topic string) (<-chan []byte, error) {
	return Default.Subscribe(topic)
}

// Unsubscribe via the default Client
func Unsubscribe(ch <-chan []byte) error {
	return Default.Unsubscribe(ch)
}

// New returns a new Client
func New(opts ...Option) Client {
	return newClient(opts...)
}
