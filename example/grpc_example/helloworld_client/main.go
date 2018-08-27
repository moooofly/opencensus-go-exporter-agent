package main

import (
	"flag"
	"log"
	"os"
	"time"

	agent "github.com/moooofly/opencensus-go-exporter-agent"
	pb "go.opencensus.io/examples/grpc/proto"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	grpcServerListenAddr = flag.String("grpc_server_listen_addr", "", "Default gPRC server listen addr.")

	// NOTE: should obtain this from $HOST_IP env
	agentIp = flag.String("agent_tcp_ip", os.Getenv("HOST_IP"),
		"The ip of TCP endport of Hunter agent, can also set with HOST_IP env.")

	agentPort = flag.String("agent_tcp_port", "12345",
		"The port of TCP endport of Hunter agent, use 12345 by default.")

	unixsockAddr = flag.String("agent_unix_addr", os.Getenv("AGENT_UNIX_ADDR"),
		"The Unix endpoint of Hunter agent, can also set with AGENT_UNIX_ADDR env. (Format: unix:///<path-to-unix-domain>)")

	// NOTE: should obtain this from $HOSTNAME env
	hostname = flag.String("hostname", os.Getenv("HOSTNAME"), "As an Attribute of span.")
)

var (
	defaultName          = "world"
	defaultTCPListenAddr = "0.0.0.0:50051"

	// obtain service_name from config file
	fakeconfig = "config.fake"
)

func main() {
	flag.Parse()

	//if *tcpAddr == "" && *unixsockAddr == "" {
	if *agentIp == "" {
		flag.Usage()
		os.Exit(0)
	}

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

	// Register the view to collect gRPC client stats.
	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		log.Fatal(err)
	}

	// Set up a connection to the server with the OpenCensus
	// stats handler to enable stats and tracing.
	info := &ocgrpc.CustomInfo{
		ServiceName: "helloworld-client" + "-" + agent.ConfigRead(fakeconfig, "cluster"),
		MethodName:  "GetUserProfile",
		RemoteKind:  "grpc",
		UID:         int64(123456),
		Source:      "web",
		HostName:    *hostname,
	}
	ch := ocgrpc.NewClientHandler(info)
	ch.StartOptions.Sampler = trace.AlwaysSample()

	var addr string
	if *grpcServerListenAddr == "" {
		addr = defaultTCPListenAddr
	} else {
		addr = *grpcServerListenAddr
	}

	conn, err := grpc.Dial(addr, grpc.WithStatsHandler(ch), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	/*
		if len(os.Args) > 1 {
			name = os.Args[1]
		}
	*/
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
