package main

import (
	"flag"
	"log"
	"strings"

	"github.com/asim/mq/go/client"
)

var (
	servers = flag.String("servers", "localhost:8081", "Comma separated list of MQ servers")
)

func main() {
	flag.Parse()

	c := client.New(client.WithServers(strings.Split(*servers, ",")...))

	ch, err := c.Subscribe("foo")
	if err != nil {
		log.Println(err)
		return
	}

	for e := range ch {
		log.Println(string(e))
	}

	log.Println("channel closed")
}
