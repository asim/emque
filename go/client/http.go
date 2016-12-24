package client

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// internal httpClient
type httpClient struct {
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

// internal select all
type all struct {
	sync.RWMutex
	servers []string
}

var (
	httpc = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	wsd = &websocket.Dialer{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
)

func publish(addr, topic string, payload []byte) error {
	url := fmt.Sprintf("%s/pub?topic=%s", addr, topic)
	rsp, err := httpc.Post(url, "application/json", bytes.NewBuffer(payload))
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
	c, _, err := wsd.Dial(url, make(http.Header))
	if err != nil {
		return err
	}

	go func() {
		select {
		case <-s.exit:
			c.Close()
		}
	}()

	go func() {
		defer s.wg.Done()

		for {
			t, p, err := c.ReadMessage()
			if err != nil || t == websocket.CloseMessage {
				c.Close()
				return
			}

			select {
			case s.ch <- p:
			case <-s.exit:
				c.Close()
				return
			}
		}
	}()

	return nil
}

func (sa *all) Get(topic string) ([]string, error) {
	sa.RLock()
	if len(sa.servers) == 0 {
		sa.RUnlock()
		return nil, errors.New("no servers")
	}
	servers := sa.servers
	sa.RUnlock()
	return servers, nil
}

func (sa *all) Set(servers ...string) error {
	sa.Lock()
	sa.servers = servers
	sa.Unlock()
	return nil
}

func (c *httpClient) run() {
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
						ips[i] = fmt.Sprintf("https://%s", ip)
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

func (c *httpClient) Close() error {
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

func (c *httpClient) Publish(topic string, payload []byte) error {
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

func (c *httpClient) Subscribe(topic string) (<-chan []byte, error) {
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

func (c *httpClient) Unsubscribe(ch <-chan []byte) error {
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

// newHTTPClient returns a http Client
func newHTTPClient(opts ...Option) *httpClient {
	options := Options{
		Selector: new(all),
		Servers:  Servers,
		Retries:  Retries,
	}

	for _, o := range opts {
		o(&options)
	}

	var servers []string

	for _, addr := range options.Servers {
		if !strings.HasPrefix(addr, "http") {
			addr = fmt.Sprintf("https://%s", addr)
		}
		servers = append(servers, addr)
	}

	// set servers
	WithServers(servers...)(&options)
	options.Selector.Set(options.Servers...)

	c := &httpClient{
		exit:        make(chan bool),
		options:     options,
		subscribers: make(map[<-chan []byte]*subscriber),
	}
	go c.run()
	return c
}
