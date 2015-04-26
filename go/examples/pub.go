package main

import (
	"log"
	"time"

	"github.com/asim/mq/go/client"
)

func main() {
	tick := time.NewTicker(time.Second)

	for _ = range tick.C {
		if err := client.Publish("foo", []byte(`bar`)); err != nil {
			log.Println(err)
			break
		}
	}
}
