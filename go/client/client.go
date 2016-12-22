package client

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// internal client
type client struct {
	exit    chan bool
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
	Close() error
	Publish(topic string, payload []byte) error
	Subscribe(topic string) (<-chan []byte, error)
	Unsubscribe(<-chan []byte) error
}

// Resolver resolves a name to a list of servers
type Resolver interface {
	Resolve(name string) ([]string, error)
}

// Selector provides a server list to publish/subscribe to
type Selector interface {
	Get(topic string) ([]string, error)
	Set(servers ...string) error
}

var (
	// The default client
	Default = newClient()
	// The default server list
	Servers = []string{"http://127.0.0.1:8081"}
	// The default number of retries
	Retries = 1
)

func newClient(opts ...Option) *client {
	options := Options{
		Selector: new(SelectAll),
		Servers:  Servers,
		Retries:  Retries,
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
	options.Selector.Set(options.Servers...)

	c := &client{
		exit:        make(chan bool),
		options:     options,
		subscribers: make(map[<-chan []byte]*subscriber),
	}
	go c.run()
	return c
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
			case s.ch <- p:
			}
		}
	}()

	return nil
}

func (c *client) run() {
	// is there a resolver?
	if c.options.Resolver == nil {
		return
	}

	t := time.NewTicker(time.Second * 30)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			var servers []string

			// iterate names
			for _, server := range c.options.Servers {
				ips, err := c.options.Resolver.Resolve(server)
				if err != nil {
					continue
				}
				for i, ip := range ips {
					if !strings.HasPrefix(ip, "http") {
						ips[i] = fmt.Sprintf("http://%s", ip)
					}
				}
				servers = append(servers, ips...)
			}

			// only set if we have servers
			if len(servers) > 0 {
				c.options.Selector.Set(servers...)
			}
		case <-c.exit:
			return
		}
	}
}

func (c *client) Close() error {
	select {
	case <-c.exit:
		return nil
	default:
		close(c.exit)
		c.Lock()
		for _, sub := range c.subscribers {
			sub.Close()
		}
		c.Unlock()
	}
	return nil
}

func (c *client) Publish(topic string, payload []byte) error {
	select {
	case <-c.exit:
		return errors.New("client closed")
	default:
	}

	servers, err := c.options.Selector.Get(topic)
	if err != nil {
		return err
	}

	var grr error
	for _, addr := range servers {
		for i := 0; i < 1+c.options.Retries; i++ {
			err := publish(addr, topic, payload)
			if err == nil {
				break
			}
			grr = err
		}
	}
	return grr
}

func (c *client) Subscribe(topic string) (<-chan []byte, error) {
	select {
	case <-c.exit:
		return nil, errors.New("client closed")
	default:
	}

	servers, err := c.options.Selector.Get(topic)
	if err != nil {
		return nil, err
	}

	ch := make(chan []byte, len(c.options.Servers)*256)

	s := &subscriber{
		ch:    ch,
		exit:  make(chan bool),
		topic: topic,
	}

	var grr error
	for _, addr := range servers {
		for i := 0; i < 1+c.options.Retries; i++ {
			err := subscribe(addr, s)
			if err == nil {
				s.wg.Add(1)
				break
			}
			grr = err
		}
	}

	return ch, grr
}

func (c *client) Unsubscribe(ch <-chan []byte) error {
	select {
	case <-c.exit:
		return errors.New("client closed")
	default:
	}

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
