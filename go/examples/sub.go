package main

import (
	"log"

	"github.com/asim/mq/go/client"
)

func main() {
	ch, err := client.Subscribe("foo")
	if err != nil {
		log.Println(err)
		return
	}

	for e := range ch {
		log.Println(string(e))
	}

	log.Println("channel closed")
}
