# MQ

MQ is a simple in-memory message broker which supports http2.

It is used mainly for testing. Purely a toy project.

## Features

- In-memory message queue
- Client side clustering
- Proxying MQ cluster
- Go client
- HTTP2
- TLS

## Usage

### Install

```shell
go get github.com/asim/mq
```

### Run

Listens on `*:8081`
```shell
mq
```

Set server address
```shell
mq --address=localhost:9091
```

Enable TLS
```shell
mq --cert_file=cert.pem --key_file=key.pem
```

### Publish

Publish via HTTP

```
curl -d "A completely arbitrary message" "http://localhost:8081/pub?topic=foo"
```

### Subscribe

Subscribe via websockets

```
curl -i -N -H "Connection: Upgrade" \
	-H "Upgrade: websocket" \
	-H "Host: localhost:8081" \
	-H "Origin:http://localhost:8081" \
	"http://localhost:8081/sub?topic=foo"
```

## Go Client

MQ provides a simple go client

### Publish

```go
package main

import (
	"time"

	"github.com/asim/mq/go/client"
)

func main() {
	tick := time.NewTicker(time.Second)

	for _ = range tick.C {
		if err := client.Publish("foo", []byte(`bar`)); err != nil {
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
		return
	}

	for e := range ch {
		log.Println(string(e))
	}
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

### MQ Proxy

MQ can be run as a proxy for an MQ cluster

```shell
mq --proxy --servers=10.0.0.1:8081,10.0.0.1:8082,10.0.0.1:8083
```
