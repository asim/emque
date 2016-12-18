package client

type Options struct {
	Servers []string
}

type Option func(o *Options)

// WithServers sets the servers used by the client
func WithServers(addrs ...string) Option {
	return func(o *Options) {
		o.Servers = addrs
	}
}
