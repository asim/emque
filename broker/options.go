package broker

import (
	"github.com/asim/emque/client"
)

type Options struct {
	Client  client.Client
	Proxy   bool
	Persist bool
}

type Option func(o *Options)

func Client(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

func Proxy(b bool) Option {
	return func(o *Options) {
		o.Proxy = b
	}
}

func Persist(b bool) Option {
	return func(o *Options) {
		o.Persist = b
	}
}
