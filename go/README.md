# Go Client

## Usage

### Publish

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

### Subscribe
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

### New Client

```go
// defaults to MQ server localhost:8081
c := client.New()
```

### Clustering

Clustering is supported on the client side. Publish/Subscribe operations are performed against all servers.
```go
c := client.New(
	client.WithServers("10.0.0.1:8081", "10.0.0.1:8082", "10.0.0.1:8083"),
)
```
