package main

//go:generate protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative  proto/mq.proto
