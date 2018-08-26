//go:generate protoc -I ../proto --go_out=plugins=grpc:../proto ../proto/helloworld.proto

package main

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
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

const port = ":50051"

var (
	// NOTE: should obtain this from $HOST_IP env
	tcpAddr      = "tcp://0.0.0.0:12345"
	unixsockAddr = "unix:///var/run/hunter-agent.sock"

	// NOTE: should obtain this from $HOSTNAME env
	hostname = "fake-server-hostname"

	// obtain service_name from config file
	fakeconfig  = "config.fake"
	serviceName = agent.ConfigRead(fakeconfig, "cluster")
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	ctx, span := trace.StartSpan(ctx, "sleep1")

	// --
	span.AddAttributes(trace.StringAttribute("service_name", "sleep2"))
	span.AddAttributes(trace.StringAttribute("method_name", "do_some_sleep"))
	span.SetName("setname")
	span.AddAttributes(trace.StringAttribute("remote_kind", "remote_kind_grpc"))
	span.AddAttributes(trace.Int64Attribute("uid", int64(123456)))
	span.AddAttributes(trace.StringAttribute("source", "source_grpc"))
	// --

	time.Sleep(time.Duration(rand.Float64() * float64(time.Second)))
	span.End()
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
	// Start z-Pages server.
	go func() {
		mux := http.NewServeMux()
		zpages.Handle(mux, "/debug")
		log.Fatal(http.ListenAndServe("0.0.0.0:8081", mux))
	}()

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

	// Register the views to collect server request count.
	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		log.Fatal(err)
	}

	view.SetReportingPeriod(15 * time.Second)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Set up a new server with the OpenCensus
	// stats handler to enable stats and tracing.
	info := &ocgrpc.CustomInfo{
		ServiceName: "helloworld-server" + "-" + serviceName,
		MethodName:  "GetUserProfile",
		RemoteKind:  "grpc",
		UID:         int64(123456),
		Source:      "web",
		HostName:    hostname,
	}
	sh := ocgrpc.NewServerHandler(info)
	sh.IsPublicEndpoint = false
	sh.StartOptions.Sampler = trace.AlwaysSample()
	s := grpc.NewServer(grpc.StatsHandler(sh))
	pb.RegisterGreeterServer(s, &server{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
