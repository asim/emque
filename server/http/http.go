package http

import (
	"crypto/tls"
	"net/http"
	"os"

	"github.com/asim/emque/server"
	"github.com/asim/emque/server/util"
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

	// tls cert and key specified
	if h.options.TLS != nil {
		cert := h.options.TLS.CertFile
		key := h.options.TLS.KeyFile
		return http.ListenAndServeTLS(address, cert, key, handler)
	}

	// generate tls config
	addr, err := util.Address(address)
	if err != nil {
		return err
	}

	cert, err := util.Certificate(addr)
	if err != nil {
		return err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	}

	l, err := tls.Listen("tcp", address, config)
	if err != nil {
		return err
	}
	defer l.Close()

	srv := &http.Server{
		Addr:      address,
		Handler:   handler,
		TLSConfig: config,
	}

	return srv.Serve(l)
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
