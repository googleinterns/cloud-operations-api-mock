package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/grpc"

	"github.com/googleinterns/cloud-operations-api-mock/server/trace"
)

const defaultPort = 8080

var port = flag.Int("port", defaultPort, "The port to run the server on.")

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("mock server failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	cloudtrace.RegisterTraceServiceServer(grpcServer, &trace.MockTraceServer{})

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("mock server failed to serve: %v", err)
	}
}
