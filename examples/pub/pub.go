package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/asim/emque/client"
)

var (
	servers = flag.String("servers", "localhost:8081", "Comma separated list of MQ servers")
)

func main() {
	flag.Parse()

	c := client.New(
		client.WithServers(strings.Split(*servers, ",")...),
	)
	tick := time.NewTicker(time.Second)

	for _ = range tick.C {
		if err := c.Publish("foo", []byte(`bar`)); err != nil {
			log.Println(err)
			break
		}
	}
}
