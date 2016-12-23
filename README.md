# MQ [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Travis CI](https://travis-ci.org/asim/mq.svg?branch=master)](https://travis-ci.org/asim/mq) 

MQ is a simple distributed in-memory message broker

## Features

- In-memory message queue
- Clustering
- Sharding
- Proxying
- Discovery
- Auto retries
- Optional: persist to file
- Command line client
- Go client library
- HTTP2 or gRPC
- TLS

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

- MQ servers are standalone servers with in-memory queues and provide a HTTP API
- MQ clients shard or cluster MQ servers by publish/subscribing to one or all servers
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

Or

```shell
docker pull chuhnk/mq
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

Use gRPC transport
```shell
mq --transport=grpc
```

### Run Proxy

MQ can be run as a proxy which includes clustering, sharding and auto retry features.

Clustering: Publish and subscribe to all MQ servers

```shell
mq --proxy --servers=10.0.0.1:8081,10.0.0.1:8082,10.0.0.1:8083
```

Sharding: Requests are sent to a single server based on topic

```shell
mq --proxy --servers=10.0.0.1:8081,10.0.0.1:8082,10.0.0.1:8083 --select=shard
```

Resolver: Use a name resolver rather than specifying server ips

```shell
mq --proxy --resolver=dns --servers=mq.proxy.dev
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

## Go Client [![GoDoc](https://godoc.org/github.com/asim/mq/go/client?status.svg)](https://godoc.org/github.com/asim/mq/go/client)

MQ provides a simple go client

```go
import "github.com/asim/mq/go/client"
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
// defaults to MQ server localhost:8081
c := client.New()
```

gRPC client

```go
c := client.NewGRPCClient()
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
c := client.New(
	client.WithServers("10.0.0.1:8081", "10.0.0.1:8082", "10.0.0.1:8083"),
	client.WithSelector(new(client.SelectShard)),
)
```
### Resolver

A name resolver can be used to discover the ip addresses of MQ servers

```go
c := client.New(
	// use the DNS resolver
	client.WithResolver(new(client.DNSResolver)),
	// specify DNS name as server
	client.WithServers("mq.proxy.local"),
)
```
