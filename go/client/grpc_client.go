package client

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/asim/mq/proto/grpc/mq"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// internal grpcClient
type grpcClient struct {
	exit    chan bool
	options Options

	sync.RWMutex
	subscribers map[<-chan []byte]*subscriber
}

func grpcPublish(addr, topic string, payload []byte) error {
	var dialOpts []grpc.DialOption

	if !strings.HasSuffix(addr, ":443") {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}

	// TODO: dial secure
	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	c := mq.NewMQClient(conn)
	_, err = c.Pub(context.TODO(), &mq.PubRequest{
		Topic:   topic,
		Payload: payload,
	})

	return err
}

func grpcSubscribe(addr string, s *subscriber) error {
	// TODO: dial secure
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	c := mq.NewMQClient(conn)
	sub, err := c.Sub(context.TODO(), &mq.SubRequest{
		Topic: s.topic,
	})
	if err != nil {
		return err
	}

	go func() {
		select {
		case <-s.exit:
			conn.Close()
		}
	}()

	go func() {
		defer s.wg.Done()

		for {
			rsp, err := sub.Recv()
			if err != nil {
				conn.Close()
				return
			}

			select {
			case s.ch <- rsp.Payload:
			case <-s.exit:
				return
			}
		}
	}()

	return nil
}

func (c *grpcClient) run() {
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

func (c *grpcClient) Close() error {
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

func (c *grpcClient) Publish(topic string, payload []byte) error {
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
			err := grpcPublish(addr, topic, payload)
			if err == nil {
				break
			}
			grr = err
		}
	}
	return grr
}

func (c *grpcClient) Subscribe(topic string) (<-chan []byte, error) {
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
			err := grpcSubscribe(addr, s)
			if err == nil {
				s.wg.Add(1)
				break
			}
			grr = err
		}
	}

	return ch, grr
}

func (c *grpcClient) Unsubscribe(ch <-chan []byte) error {
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

// NewGRPCClient returns a grpc Client
func NewGRPCClient(opts ...Option) *grpcClient {
	options := Options{
		Selector: new(SelectAll),
		Servers:  Servers,
		Retries:  Retries,
	}

	for _, o := range opts {
		o(&options)
	}

	// set servers
	options.Selector.Set(options.Servers...)

	c := &grpcClient{
		exit:        make(chan bool),
		options:     options,
		subscribers: make(map[<-chan []byte]*subscriber),
	}
	go c.run()
	return c
}
