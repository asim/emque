package client

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
	Default = New()
	// The default server list
	Servers = []string{"http://127.0.0.1:8081"}
	// The default number of retries
	Retries = 1
)

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
	return newHTTPClient(opts...)
}
