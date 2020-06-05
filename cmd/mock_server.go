// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"log"
	"net"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"

	"github.com/googleinterns/cloud-operations-api-mock/server/metric"
	"github.com/googleinterns/cloud-operations-api-mock/server/trace"
)

const (
	defaultAddress = "localhost:8080"
)

var (
	address = flag.String("address", defaultAddress,
		"The address to run the server on, of the form <host>:<port>.")
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatalf("mock server failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	cloudtrace.RegisterTraceServiceServer(grpcServer, &trace.MockTraceServer{})
	monitoring.RegisterMetricServiceServer(grpcServer, &metric.MockMetricServer{})

	log.Printf("Listening on %s\n", *address)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("mock server failed to serve: %v", err)
	}
}
