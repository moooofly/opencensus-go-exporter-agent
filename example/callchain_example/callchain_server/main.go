package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	agent "github.com/moooofly/opencensus-go-exporter-hunter"
	pb "github.com/moooofly/opencensus-go-exporter-hunter/example/callchain_example/proto"
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

	defaultPort = "50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	var wg sync.WaitGroup

	var err error
	if rand.Float32() < in.GetErrorRate() {
		err = errors.New("trigger error by probability")
	} else {
		err = nil
	}

	nodes := in.GetNodes()
	if nodes == "" {
		return &pb.HelloReply{Message: "[END] Hello " + in.Name}, err
	}

	info := ocgrpc.NewClientCustomInfo(
		agent.ConfigRead(*configPath, "cluster"),
		*hostname,
	)

	ch := ocgrpc.NewClientHandler(info)

	// FIXME: is it necessary?
	//ch.StartOptions.Sampler = trace.AlwaysSample()

	if strings.Index(nodes, "|") == -1 {
		log.Printf("nodes to call: %q\n", nodes)

		dests := genServiceName(nodes)
		log.Println(dests)

		for _, d := range dests {
			wg.Add(1)
			go call(ctx, &wg, d, ch, "", in)
		}
	} else {
		split := strings.SplitN(nodes, "|", 2)
		log.Printf("nodes to call: %q\n", split[0])
		log.Printf("nodes to deliver: %q\n", split[1])

		dests := genServiceName(split[0])
		log.Println(dests)

		for _, d := range dests {
			wg.Add(1)
			go call(ctx, &wg, d, ch, split[1], in)
		}
	}

	wg.Wait()

	return &pb.HelloReply{Message: "Hello " + in.Name}, err
}

func main() {
	flag.Parse()

	//if *agentIp == "" && *unixsockAddr == "" {
	if *agentIp == "" {
		log.Println("agent_tcp_ip must not be empty.")
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
	defer exporter.Flush()

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
		addr = "0.0.0.0:" + defaultPort
	} else {
		addr = "0.0.0.0:" + *grpcServerListenPort
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Set up a new server with the OpenCensus
	// stats handler to enable stats and tracing.
	info := ocgrpc.NewServerCustomInfo(
		agent.ConfigRead(*configPath, "cluster"),
		*hostname,
	)
	sh := ocgrpc.NewServerHandler(info)
	sh.IsPublicEndpoint = false

	// FIXME:
	// If remote parent (client) set with specific Sampler, server side dose not need to set again.
	// If changing sample rate is your option, just do it here.
	//sh.StartOptions.Sampler = trace.AlwaysSample()

	s := grpc.NewServer(grpc.StatsHandler(sh))
	pb.RegisterGreeterServer(s, &server{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func call(ctx context.Context, wg *sync.WaitGroup, addr string, ch *ocgrpc.ClientHandler, left string, in *pb.HelloRequest) {
	defer wg.Done()

	conn, err := grpc.Dial(addr, grpc.WithStatsHandler(ch), grpc.WithInsecure())
	if err != nil {
		log.Printf("Cannot connect: %v", err)
		return
	}
	defer conn.Close()

	c := pb.NewGreeterClient(conn)

	r, err := c.SayHello(ctx, &pb.HelloRequest{
		Name:      in.GetName(),
		Nodes:     left,
		ErrorRate: in.GetErrorRate(),
	})
	if err != nil {
		log.Printf("Could not greet: %v", err)
	} else {
		log.Printf("Greeting: %s", r.Message)
	}
}

func genServiceName(in string) []string {
	if in == "" {
		return nil
	}
	ns := strings.Split(in, ",")

	out := make([]string, 0, len(ns))
	for _, n := range ns {
		out = append(out, fmt.Sprintf("%s%s:%s", "hunterdemo-spider-node", n, defaultPort))
	}

	return out
}
