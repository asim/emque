package client

type Options struct {
	// Number of retry attempts
	Retries int
	// Server list
	Servers []string
}

type Option func(o *Options)

// WithServers sets the servers used by the client
func WithServers(addrs ...string) Option {
	return func(o *Options) {
		o.Servers = addrs
	}
}

// WithRetries sets the number of retry attempts
func WithRetries(i int) Option {
	return func(o *Options) {
		o.Retries = i
	}
}
