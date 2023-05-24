# Go Client [![GoDoc](https://godoc.org/github.com/asim/emque/client?status.svg)](https://godoc.org/github.com/asim/emque/client)

## Usage

### Publish

```go
package main

import (
	"log"
	"time"

	"github.com/asim/emque/client"
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

	"github.com/asim/emque/client"
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

gRPC client

```go
import "github.com/asim/emque/client/grpc"

c := grpc.New()
```

### Clustering

Clustering is supported on the client side. Publish/Subscribe operations are performed against all servers.

```go
c := client.New(
	client.WithServers("10.0.0.1:8081", "10.0.0.1:8082", "10.0.0.1:8083"),
)
```

### Sharding

Sharding is supported via client much like gomemcache. Publish/Subscribe operations are performed against a single server.

```go
import "github.com/asim/emque/client/selector"

c := client.New(
	client.WithServers("10.0.0.1:8081", "10.0.0.1:8082", "10.0.0.1:8083"),
	client.WithSelector(new(selector.Shard)),
)
```

### Resolver

A name resolver can be used to discover the ip addresses of MQ servers

```go
import "github.com/asim/emque/client/resolver"

c := client.New(
	// use the DNS resolver
	client.WithResolver(new(resolver.DNS)),
	// specify DNS name as server
	client.WithServers("mq.proxy.local"),
)
```
