package broker

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/asim/mq/go/client"
)

var (
	Default Broker = newBroker()
)

// internal broker
type broker struct {
	options *Options

	sync.RWMutex
	topics map[string][]chan []byte

	mtx       sync.RWMutex
	persisted map[string]bool
}

// internal message for persistence
type message struct {
	Timestamp int64  `json:"timestamp"`
	Topic     string `json:"topic"`
	Payload   []byte `json:"payload"`
}

// Broker is the message broker
type Broker interface {
	Publish(topic string, payload []byte) error
	Subscribe(topic string) (<-chan []byte, error)
	Unsubscribe(topic string, sub <-chan []byte) error
}

func newBroker(opts ...Option) *broker {
	options := &Options{
		Client: client.New(),
	}

	for _, o := range opts {
		o(options)
	}

	return &broker{
		options:   options,
		topics:    make(map[string][]chan []byte),
		persisted: make(map[string]bool),
	}
}

func publish(payload []byte, subscribers []chan []byte) {
	n := len(subscribers)
	c := 1

	// increase concurrency if there are many subscribers
	switch {
	case n > 1000:
		c = 3
	case n > 100:
		c = 2
	}

	// publisher function
	pub := func(start int) {
		// iterate the subscribers
		for j := start; j < n; j += c {
			select {
			// push the payload to subscriber
			case subscribers[j] <- payload:
			// only wait 5 milliseconds for subscriber
			case <-time.After(time.Millisecond * 5):
			}
		}
	}

	// concurrent publish
	for i := 0; i < c; i++ {
		go pub(i)
	}
}

func (b *broker) persist(topic string) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.persisted[topic] {
		return nil
	}

	ch, err := b.Subscribe(topic)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(topic+".mq", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		return err
	}

	go func() {
		var pending []byte
		newline := []byte{'\n'}
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			select {
			case p := <-ch:
				b, err := json.Marshal(&message{
					Timestamp: time.Now().UnixNano(),
					Topic:     topic,
					Payload:   p,
				})
				if err != nil {
					continue
				}
				pending = append(pending, b...)
				pending = append(pending, newline...)
			case <-t.C:
				if len(pending) == 0 {
					continue
				}
				f.Write(pending)
				pending = nil
			}
		}
	}()

	b.persisted[topic] = true

	return nil
}

func (b *broker) Publish(topic string, payload []byte) error {
	if b.options.Proxy {
		return b.options.Client.Publish(topic, payload)
	}

	b.RLock()
	subscribers, ok := b.topics[topic]
	b.RUnlock()
	if !ok {
		// persist?
		if !b.options.Persist {
			return nil
		}
		if err := b.persist(topic); err != nil {
			return err
		}
	}

	publish(payload, subscribers)

	return nil
}

func (b *broker) Subscribe(topic string) (<-chan []byte, error) {
	if b.options.Proxy {
		return b.options.Client.Subscribe(topic)
	}

	ch := make(chan []byte, 100)
	b.Lock()
	b.topics[topic] = append(b.topics[topic], ch)
	b.Unlock()
	return ch, nil
}

func (b *broker) Unsubscribe(topic string, sub <-chan []byte) error {
	if b.options.Proxy {
		return b.options.Client.Unsubscribe(sub)
	}

	b.RLock()
	subscribers, ok := b.topics[topic]
	b.RUnlock()

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

	b.Lock()
	b.topics[topic] = subs
	b.Unlock()
	return nil
}

func Publish(topic string, payload []byte) error {
	return Default.Publish(topic, payload)
}

func Subscribe(topic string) (<-chan []byte, error) {
	return Default.Subscribe(topic)
}

func Unsubscribe(topic string, sub <-chan []byte) error {
	return Default.Unsubscribe(topic, sub)
}

func New(opts ...Option) *broker {
	return newBroker(opts...)
}
