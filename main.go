package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/asim/emque/broker"
	mqclient "github.com/asim/emque/client"
	mqgrpc "github.com/asim/emque/client/grpc"
	mqresolver "github.com/asim/emque/client/resolver"
	mqselector "github.com/asim/emque/client/selector"
	"github.com/asim/emque/server"
	grpcsrv "github.com/asim/emque/server/grpc"
	httpsrv "github.com/asim/emque/server/http"
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
	interactive = flag.Bool("i", false, "Interactive client mode")
	client      = flag.Bool("client", false, "Run the MQ client")
	publish     = flag.Bool("publish", false, "Publish via the MQ client")
	subscribe   = flag.Bool("subscribe", false, "Subscribe via the MQ client")
	topic       = flag.String("topic", "", "Topic for client to publish or subscribe to")

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

	if (*client || *interactive) && len(*servers) == 0 {
		*servers = "localhost:8081"
	}

	var bclient mqclient.Client
	var selecter mqclient.Selector
	var resolvor mqclient.Resolver

	switch *selector {
	case "shard":
		selecter = new(mqselector.Shard)
	default:
		selecter = new(mqselector.All)
	}

	switch *resolver {
	case "dns":
		resolvor = new(mqresolver.DNS)
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
		bclient = mqgrpc.New(options...)
	default:
		bclient = mqclient.New(options...)
	}

	broker.Default = broker.New(
		broker.Client(bclient),
		broker.Persist(*persist),
		broker.Proxy(*client || *proxy || *interactive),
	)
}

func cli() {
	wg := sync.WaitGroup{}
	p := make(chan []byte, 1000)
	d := map[string]time.Time{}
	ttl := time.Millisecond * 10
	tick := time.NewTicker(time.Second * 5)

	// process publish
	if *publish || *interactive {
		wg.Add(1)
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				if *interactive {
					p <- scanner.Bytes()
				}
				broker.Publish(*topic, scanner.Bytes())
			}
			wg.Done()
		}()
	}

	// subscribe?
	if !(*subscribe || *interactive) {
		wg.Wait()
		return
	}

	// process subscribe
	ch, err := broker.Subscribe(*topic)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer broker.Unsubscribe(*topic, ch)

	for {
		select {
		// process sub event
		case b := <-ch:
			// skip if deduped
			if t, ok := d[string(b)]; ok && time.Since(t) < ttl {
				continue
			}
			d[string(b)] = time.Now()
			fmt.Println(string(b))
		// add dedupe entry
		case b := <-p:
			d[string(b)] = time.Now()
		// flush deduper
		case <-tick.C:
			d = map[string]time.Time{}
		}
	}

	wg.Wait()
}

func main() {
	// handle client
	if *client || *interactive {
		cli()
		return
	}

	// cleanup broker
	defer broker.Default.Close()

	options := []server.Option{
		server.WithAddress(*address),
	}

	// proxy enabled
	if *proxy {
		log.Println("Proxy enabled")
	}

	// tls enabled
	if len(*cert) > 0 && len(*key) > 0 {
		log.Println("TLS Enabled")
		options = append(options, server.WithTLS(*cert, *key))
	}

	var server server.Server

	// now serve the transport
	switch *transport {
	case "grpc":
		log.Println("GRPC transport enabled")
		server = grpcsrv.New(options...)
	default:
		log.Println("HTTP transport enabled")
		server = httpsrv.New(options...)
	}

	log.Println("MQ listening on", *address)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
