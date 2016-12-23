package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/asim/mq/broker"
	mqclient "github.com/asim/mq/go/client"
	"github.com/asim/mq/handler"
	"github.com/asim/mq/proto/grpc/mq"
	"github.com/gorilla/handlers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	address = flag.String("address", ":8081", "MQ server address")
	cert    = flag.String("cert_file", "", "TLS certificate file")
	key     = flag.String("key_file", "", "TLS key file")

	// server persist to file
	persist = flag.Bool("persist", false, "Persist messages to [topic].mq file per topic")

	// proxy flags
	proxy   = flag.Bool("proxy", false, "Proxy for an MQ cluster")
	retries = flag.Int("retries", 1, "Number of retries for publish or subscribe")
	servers = flag.String("servers", "", "Comma separated MQ cluster list used by Proxy")

	// client flags
	client    = flag.Bool("client", false, "Run the MQ client")
	publish   = flag.Bool("publish", false, "Publish via the MQ client")
	subscribe = flag.Bool("subscribe", false, "Subscribe via the MQ client")
	topic     = flag.String("topic", "", "Topic for client to publish or subscribe to")

	// select strategy
	selector = flag.String("select", "all", "Server select strategy. Supports all, shard")
	// resolver for discovery
	resolver = flag.String("resolver", "ip", "Server resolver for discovery. Supports ip, dns")
	// transport http or grpc
	transport = flag.String("transport", "http", "Transport for communication. Support http, grpc")
)

func init() {
	flag.Parse()

	if *proxy && *client {
		log.Fatal("Client and proxy flags cannot be specified together")
	}

	if *proxy && len(*servers) == 0 {
		log.Fatal("Proxy enabled without MQ server list")
	}

	if *client && len(*topic) == 0 {
		log.Fatal("Topic not specified")
	}

	if *client && !*publish && !*subscribe {
		log.Fatal("Specify whether to publish or subscribe")
	}

	if *client && len(*servers) == 0 {
		log.Fatal("Client specified without MQ server list")
	}

	var bclient mqclient.Client
	var selecter mqclient.Selector
	var resolvor mqclient.Resolver

	switch *selector {
	case "shard":
		selecter = new(mqclient.SelectShard)
	default:
		selecter = new(mqclient.SelectAll)
	}

	switch *resolver {
	case "dns":
		resolvor = new(mqclient.DNSResolver)
	default:
	}

	options := []mqclient.Option{
		mqclient.WithResolver(resolvor),
		mqclient.WithSelector(selecter),
		mqclient.WithServers(strings.Split(*servers, ",")...),
		mqclient.WithRetries(*retries),
	}

	switch *transport {
	case "grpc":
		bclient = mqclient.NewGRPCClient(options...)
	default:
		bclient = mqclient.NewHTTPClient(options...)
	}

	broker.Default = broker.New(
		broker.Client(bclient),
		broker.Persist(*persist),
		broker.Proxy(*client || *proxy),
	)
}

func cli() {
	// process publish
	if *publish {
		// scan till EOF
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			broker.Publish(*topic, scanner.Bytes())
		}
	}

	// subscribe?
	if !*subscribe {
		return
	}

	// process subscribe
	ch, err := broker.Subscribe(*topic)
	if err != nil {
		fmt.Println(err)
		return
	}

	for e := range ch {
		fmt.Println(string(e))
	}
}

func httpServer() {
	// MQ Handlers
	http.HandleFunc("/pub", handler.Pub)
	http.HandleFunc("/sub", handler.Sub)

	// logging handler
	handler := handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)

	// tls enabled
	if len(*cert) > 0 && len(*key) > 0 {
		if err := http.ListenAndServeTLS(*address, *cert, *key, handler); err != nil {
			log.Fatal(err)
		}
	}

	// plain server
	if err := http.ListenAndServe(*address, handler); err != nil {
		log.Fatal(err)
	}
}

func grpcServer() {
	l, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatal(err)
	}

	var opts []grpc.ServerOption

	// tls enabled
	if len(*cert) > 0 && len(*key) > 0 {
		creds, err := credentials.NewServerTLSFromFile(*cert, *key)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// new grpc server
	srv := grpc.NewServer(opts...)

	// register MQ server
	mq.RegisterMQServer(srv, new(handler.GRPC))

	// serve
	if err := srv.Serve(l); err != nil {
		log.Fatal(err)
	}
}

func main() {
	// handle client
	if *client {
		cli()
		return
	}

	// cleanup broker
	defer broker.Default.Close()

	// proxy enabled
	if *proxy {
		log.Println("Proxy enabled")
	}

	// tls enabled
	if len(*cert) > 0 && len(*key) > 0 {
		log.Println("TLS Enabled")
	}

	// now serve the transport
	switch *transport {
	case "grpc":
		log.Println("GRPC transport enabled")
		log.Println("MQ listening on", *address)
		grpcServer()
	default:
		log.Println("HTTP transport enabled")
		log.Println("MQ listening on", *address)
		httpServer()
	}
}
