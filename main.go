package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/asim/mq/broker"
	mqclient "github.com/asim/mq/go/client"
	"github.com/asim/mq/handler"
	"github.com/gorilla/handlers"
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

	var selecter mqclient.Selector

	switch *selector {
	case "shard":
		selecter = new(mqclient.SelectShard)
	default:
		selecter = new(mqclient.SelectAll)
	}

	broker.Default = broker.New(
		broker.Client(mqclient.New(
			mqclient.WithSelector(selecter),
			mqclient.WithServers(strings.Split(*servers, ",")...),
			mqclient.WithRetries(*retries),
		)),
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

func main() {
	// handle client
	if *client {
		cli()
		return
	}

	// MQ Handlers
	http.HandleFunc("/pub", handler.Pub)
	http.HandleFunc("/sub", handler.Sub)

	// logging handler
	handler := handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)

	// proxy enabled
	if *proxy {
		log.Println("Proxy enabled")
	}

	// tls enabled
	if len(*cert) > 0 && len(*key) > 0 {
		log.Println("TLS Enabled")
		log.Println("MQ listening on", *address)
		if err := http.ListenAndServeTLS(*address, *cert, *key, handler); err != nil {
			log.Fatal(err)
		}
		return
	}

	log.Println("MQ listening on", *address)
	if err := http.ListenAndServe(*address, handler); err != nil {
		log.Fatal(err)
	}
}
