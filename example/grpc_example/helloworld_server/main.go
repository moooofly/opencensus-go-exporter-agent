//go:generate protoc -I ../proto --go_out=plugins=grpc:../proto ../proto/helloworld.proto

package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	agent "github.com/moooofly/opencensus-go-exporter-agent"
	pb "go.opencensus.io/examples/grpc/proto"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	grpcServerListenPort = flag.String("grpc_server_listen_port", "", "Default gPRC server listen port.")

	// NOTE: should obtain this from $HOST_IP env
	agentIp = flag.String("agent_tcp_ip", os.Getenv("HOST_IP"),
		"The ip of TCP endport of Hunter agent, can also set with HOST_IP env.")

	agentPort = flag.String("agent_tcp_port", "12345",
		"The port of TCP endport of Hunter agent, use 12345 by default.")

	unixsockAddr = flag.String("agent_unix_addr", os.Getenv("AGENT_UNIX_ADDR"),
		"The Unix endpoint of Hunter agent, can also set with AGENT_UNIX_ADDR env. (Format: unix:///<path-to-unix-domain>)")

	// NOTE: should obtain this from $HOSTNAME env
	hostname = flag.String("hostname", os.Getenv("HOSTNAME"), "As an Attribute of span.")

	//fakeconfig = "config.fake"
	configPath = flag.String("configPath", agent.DefaultConfigPath, "Config file from which get 'cluster' item.")

	defaultTCPListenPort = "50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	/*
		ctx, span := trace.StartSpan(ctx, "sleep1")

		span.AddAttributes(trace.StringAttribute("service_name", "sleep2"))
		span.AddAttributes(trace.StringAttribute("method_name", "do_some_sleep"))
		span.SetName("setname")
		span.AddAttributes(trace.StringAttribute("remote_kind", "remote_kind_grpc"))
		span.AddAttributes(trace.Int64Attribute("uid", int64(123456)))
		span.AddAttributes(trace.StringAttribute("source", "source_grpc"))

		time.Sleep(time.Duration(rand.Float64() * float64(time.Second)))
		span.End()
	*/

	rand.Seed(time.Now().UnixNano())
	d := time.Duration(rand.Intn(3)) * time.Second
	time.Sleep(d)

	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
	flag.Parse()

	//if *agentIp == "" && *unixsockAddr == "" {
	if *agentIp == "" {
		flag.Usage()
		os.Exit(0)
	}

	// Start z-Pages server.
	go func() {
		mux := http.NewServeMux()
		zpages.Handle(mux, "/debug")
		log.Fatal(http.ListenAndServe("0.0.0.0:8081", mux))
	}()

	addrs := make(map[string]string, 2)
	addrs["tcp"] = *agentIp + ":" + *agentPort
	//addrs["unix"] = strings.TrimPrefix(*unixsockAddr, "unix://")

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

	// Register the views to collect server request count.
	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		log.Fatal(err)
	}

	view.SetReportingPeriod(15 * time.Second)

	var addr string
	if *grpcServerListenPort == "" {
		addr = "0.0.0.0:" + defaultTCPListenPort
	} else {
		addr = "0.0.0.0:" + *grpcServerListenPort
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Set up a new server with the OpenCensus
	// stats handler to enable stats and tracing.
	info := &ocgrpc.CustomInfo{
		ServiceName: agent.ConfigRead(*configPath, "cluster"),
		RemoteKind:  "grpc",
		UID:         int64(123456),
		Source:      "web",
		HostName:    *hostname,
	}
	sh := ocgrpc.NewServerHandler(info)
	sh.IsPublicEndpoint = false

	// FIXME:
	// if remote parent with specific Sampler, server side should not set this (by tony)
	// but not work as expect, so roll back
	sh.StartOptions.Sampler = trace.AlwaysSample()
	s := grpc.NewServer(grpc.StatsHandler(sh))
	pb.RegisterGreeterServer(s, &server{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
