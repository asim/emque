# MQ [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Travis CI](https://travis-ci.org/asim/mq.svg?branch=master)](https://travis-ci.org/asim/mq) 

MQ is a simple in-memory message broker

## Features

- In-memory message queue
- Optionally persist to file
- Client side clustering
- Proxying MQ cluster
- Command line client
- Go client library
- HTTP2
- TLS

## Architecture

- MQ servers are standalone servers with in-memory queues and provide a HTTP API
- MQ clients cluster MQ servers by publish/subscribing to all servers
- MQ proxies use the go client to cluster MQ servers and provide a unified HTTP API

Because of this simplistic architecture, proxies and servers can be chained to build message pipelines.

<p align="center">
  <img src="mq.png" />
</p>

## Usage

### Install

```shell
go get github.com/asim/mq
```

Or with Docker

```shell
docker run chuhnk/mq
```

### Run Server

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

Persist to file per topic
```shell
mq --persist
```

### Run Proxy

MQ can be run as a proxy. It will publish or subscribe to all MQ servers.

```shell
mq --proxy --servers=10.0.0.1:8081,10.0.0.1:8082,10.0.0.1:8083
```

### Run Client

Publish

```shell
echo "A completely arbitrary message" | mq --client --topic=foo --publish --servers=localhost:8081
```

Subscribe

```shell
mq --client --topic=foo --subscribe --servers=localhost:8081
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

