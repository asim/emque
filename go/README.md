# Go Library to connect to MQ

## Publish Example
```go
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
```

## Subscribe Example
```go
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
```
