package main

import (
	"context"
	"log"
	"net"

	"quotes/api"
	"quotes/pkg/onederx"
	"quotes/pkg/rpc"

	"google.golang.org/grpc"
)

func main() {
	onederx := onederx.NewSource()
	onederx.Start(context.Background())

	service := rpc.NewService()
	service.AddSource(onederx)
	server := grpc.NewServer()
	api.RegisterQuotesServer(server, service)

	lsn, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting server on %s", lsn.Addr().String())
	if err := server.Serve(lsn); err != nil {
		log.Fatal(err)
	}
}
