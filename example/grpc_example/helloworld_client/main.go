// Copyright 2017, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log"
	"os"
	"strings"
	"time"

	agent "github.com/moooofly/opencensus-go-exporter-agent"
	pb "go.opencensus.io/examples/grpc/proto"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
)

// FIXME: hardcode
var (
	tcpAddr      = "tcp://0.0.0.0:12345"
	unixsockAddr = "unix:///var/run/hunter-agent.sock"
)

func main() {

	addrs := make(map[string]string, 2)
	addrs["tcp"] = strings.TrimPrefix(tcpAddr, "tcp://")
	addrs["unix"] = strings.TrimPrefix(unixsockAddr, "unix://")

	exporter, err := agent.NewExporter(
		agent.Addrs(addrs),
		//agent.Logger(logger),
	)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	// Register stats and trace exporters to export
	// the collected data.
	view.RegisterExporter(exporter)
	trace.RegisterExporter(exporter)

	// Register the view to collect gRPC client stats.
	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		log.Fatal(err)
	}

	// Set up a connection to the server with the OpenCensus
	// stats handler to enable stats and tracing.
	info := &ocgrpc.CustomInfo{
		"helloworld-client-grpc",
		"GetUserProfile",
		"grpc",
		int64(123456),
		"web",
	}
	ch := ocgrpc.NewClientHandler(info)
	ch.StartOptions.Sampler = trace.AlwaysSample()
	conn, err := grpc.Dial(address, grpc.WithStatsHandler(ch), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	view.SetReportingPeriod(15 * time.Second)
	for {
		r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
		if err != nil {
			log.Printf("Could not greet: %v", err)
		} else {
			log.Printf("Greeting: %s", r.Message)
		}
		time.Sleep(2 * time.Second)
	}
}
