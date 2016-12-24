package http

import (
	"net/http"
	"os"

	"github.com/asim/mq/server"
	"github.com/gorilla/handlers"
)

type httpServer struct {
	options *server.Options
}

func (h *httpServer) Run() error {
	// MQ Handlers
	http.HandleFunc("/pub", pub)
	http.HandleFunc("/sub", sub)

	// logging handler
	handler := handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)
	address := h.options.Address

	// tls enabled
	if h.options.TLS != nil {
		cert := h.options.TLS.CertFile
		key := h.options.TLS.KeyFile
		return http.ListenAndServeTLS(address, cert, key, handler)
	}

	return http.ListenAndServe(address, handler)
}

func New(opts ...server.Option) *httpServer {
	options := new(server.Options)
	for _, o := range opts {
		o(options)
	}
	return &httpServer{
		options: options,
	}
}
