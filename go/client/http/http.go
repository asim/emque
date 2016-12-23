package http

import (
	"github.com/asim/mq/go/client"
)

// New returns a http client
func New(opts ...client.Option) client.Client {
	return client.New(opts...)
}
