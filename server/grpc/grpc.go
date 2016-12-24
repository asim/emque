package grpc

import (
	"net"

	"github.com/asim/mq/handler"
	"github.com/asim/mq/proto/grpc/mq"
	"github.com/asim/mq/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type grpcServer struct {
	options *server.Options
}

func (g *grpcServer) Run() error {
	l, err := net.Listen("tcp", g.options.Address)
	if err != nil {
		return err
	}

	var opts []grpc.ServerOption

	// tls enabled
	if g.options.TLS != nil {
		creds, err := credentials.NewServerTLSFromFile(
			g.options.TLS.CertFile,
			g.options.TLS.KeyFile,
		)
		if err != nil {
			return err
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// new grpc server
	srv := grpc.NewServer(opts...)

	// register MQ server
	mq.RegisterMQServer(srv, new(handler.GRPC))

	// serve
	return srv.Serve(l)
}

func New(opts ...server.Option) *grpcServer {
	options := new(server.Options)
	for _, o := range opts {
		o(options)
	}
	return &grpcServer{
		options: options,
	}
}
