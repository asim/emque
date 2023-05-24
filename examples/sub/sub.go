package main

import (
	"flag"
	"log"
	"strings"

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

	ch, err := c.Subscribe("foo")
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Unsubscribe(ch)

	for {
		select {
		case e := <-ch:
			log.Println(string(e))
		}
	}
}
