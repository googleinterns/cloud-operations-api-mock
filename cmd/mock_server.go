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
	"html/template"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	mocktrace "github.com/googleinterns/cloud-operations-api-mock/api"

	"github.com/googleinterns/cloud-operations-api-mock/server/metric"
	"github.com/googleinterns/cloud-operations-api-mock/server/trace"
	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
)

const (
	defaultAddress = "localhost:8080"
)

var (
	address = flag.String("address", defaultAddress,
		"The address to run the server on, of the form <host>:<port>")
	summary = flag.Bool("summary", false,
		"If flag is set, a summary page HTML file will be generated")
)

func main() {
	flag.Parse()
	startStandaloneServer()
}

// startStandaloneServer spins up the mock server, and listens for requests.
// Upon detecting a kill signal, it will shutdown and optionally print out a table of results.
// Flags:
// -address=<host:port> will start the server at the given address
// -summary will tell the server to generate an HTML file containing the data received.
func startStandaloneServer() {
	lis, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatalf("mock server failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	mockTraceServer := trace.NewMockTraceServer()
	cloudtrace.RegisterTraceServiceServer(grpcServer, mockTraceServer)
	mocktrace.RegisterMockTraceServiceServer(grpcServer, mockTraceServer)
	monitoring.RegisterMetricServiceServer(grpcServer, metric.NewMockMetricServer())

	log.Printf("Listening on %s\n", lis.Addr().String())

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// Allow the summary to be fully written before exiting.
	finish := make(chan bool)

	go func() {
		<-sig
		grpcServer.GracefulStop()
		if *summary {
			writeSummaryPage(mockTraceServer.ResultTable())
		}
		finish <- true
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("mock server failed to serve: %v", err)
	}
	<-finish
}

// writeSummaryPage creates summary.html from the results and the template HTML.
func writeSummaryPage(results *[]*cloudtrace.Span) {
	outputFile, err := os.Create("summary.html")
	if err != nil {
		panic(err)
	}
	t := template.Must(template.ParseFiles("../static/summary_template.html"))
	err = t.Execute(outputFile, results)
	if err != nil {
		panic(err)
	}
}
