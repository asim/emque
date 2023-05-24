# Emque [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Go Report Card](https://goreportcard.com/badge/asim/emque)](https://goreportcard.com/report/github.com/asim/emque)

Emque is an in-memory message broker

## Features

- In-memory message broker
- HTTP or gRPC transport
- Clustering
- Sharding
- Proxying
- Discovery
- Auto retries
- TLS support
- Command line interface
- Interactive prompt
- Go client library

Emque generates a self signed certificate by default if no TLS config is specified

## API

Publish
```
/pub?topic=string	publish payload as body
```

Subscribe
```
/sub?topic=string	subscribe as websocket
```

## Architecture

- Emque servers are standalone servers with in-memory queues and provide a HTTP API
- Emque clients shard or cluster Emque servers by publish/subscribing to one or all servers
- Emque proxies use the go client to cluster Emque servers and provide a unified HTTP API

<p align="center">
  <img src="emque.png" />
</p>

Because of this simplistic architecture, proxies and servers can be chained to build message pipelines

## Usage

### Install

```shell
go get github.com/asim/emque
```

### Run Server

Listens on `*:8081`
```shell
emque
```

Set server address
```shell
emque --address=localhost:9091
```

Enable TLS
```shell
emque --cert_file=cert.pem --key_file=key.pem
```

Persist to file per topic
```shell
emque --persist
```

Use gRPC transport
```shell
emque --transport=grpc
```

### Run Proxy

Emque can be run as a proxy which includes clustering, sharding and auto retry features.

Clustering: Publish and subscribe to all Emque servers

```shell
emque --proxy --servers=10.0.0.1:8081,10.0.0.1:8082,10.0.0.1:8083
```

Sharding: Requests are sent to a single server based on topic

```shell
emque --proxy --servers=10.0.0.1:8081,10.0.0.1:8082,10.0.0.1:8083 --select=shard
```

Resolver: Use a name resolver rather than specifying server ips

```shell
emque --proxy --resolver=dns --servers=emque.proxy.dev
```

### Run Client

Publish

```shell
echo "A completely arbitrary message" | emque --client --topic=foo --publish --servers=localhost:8081
```

Subscribe

```shell
emque --client --topic=foo --subscribe --servers=localhost:8081
``` 

Interactive mode
```shell
emque -i --topic=foo
```

### Publish

Publish via HTTP

```
curl -k -d "A completely arbitrary message" "https://localhost:8081/pub?topic=foo"
```

### Subscribe

Subscribe via websockets

```
curl -k -i -N -H "Connection: Upgrade" \
	-H "Upgrade: websocket" \
	-H "Host: localhost:8081" \
	-H "Origin:http://localhost:8081" \
	-H "Sec-Websocket-Version: 13" \
	-H "Sec-Websocket-Key: Emque" \
	"https://localhost:8081/sub?topic=foo"
```

## Go Client [![GoDoc](https://godoc.org/github.com/asim/emque/client?status.svg)](https://godoc.org/github.com/asim/emque/client)

Emque provides a simple go client

```go
import "github.com/asim/emque/client"
```

### Publish

```go
// publish to topic foo
err := client.Publish("foo", []byte(`bar`))
```

### Subscribe

```go
// subscribe to topic foo
ch, err := client.Subscribe("foo")
if err != nil {
	return
}

data := <-ch
```

### New Client

```go
// defaults to Emque server localhost:8081
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

A name resolver can be used to discover the ip addresses of Emque servers

```go
import "github.com/asim/emque/client/resolver"

c := client.New(
	// use the DNS resolver
	client.WithResolver(new(resolver.DNS)),
	// specify DNS name as server
	client.WithServers("emque.proxy.local"),
)
```
