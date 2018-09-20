package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	agent "github.com/moooofly/opencensus-go-exporter-hunter"
	pb "github.com/moooofly/opencensus-go-exporter-hunter/example/callchain_example/proto"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
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

	// TODO: generate a random callchain string as default value
	nodes = flag.String("nodes", "1,2|3|4,5,6|7,8|9,10", "Nodes specified by special formats to simulate callchain.")
)

var (
	defaultName = "callchain"
	defaultPort = "50051"
	//defaultTCPListenAddr = "0.0.0.0" + defaultPort
	defaultTCPListenAddr = fmt.Sprintf("0.0.0.0:%s", defaultPort)
)

func main() {
	flag.Parse()

	//if *tcpAddr == "" && *unixsockAddr == "" {
	if *agentIp == "" {
		log.Println("agent_tcp_ip must not be empty.")
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
	defer exporter.Flush()

	// Register stats and trace exporters to export
	// the collected data.
	view.RegisterExporter(exporter)
	trace.RegisterExporter(exporter)

	// Register the view to collect gRPC client stats.
	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		log.Fatal(err)
	}

	// Contact the server and print out its response.
	view.SetReportingPeriod(15 * time.Second)

	// Set up a connection to the server with the OpenCensus
	// stats handler to enable stats and tracing.
	info := ocgrpc.NewClientCustomInfo(
		agent.ConfigRead(*configPath, "cluster"),
		*hostname,
	)

	ch := ocgrpc.NewClientHandler(info)
	ch.StartOptions.Sampler = trace.AlwaysSample()

	// k8s service name format: hunterdemo-spider-node{1-10}
	// parse nodes to get addrs to be called.
	if *nodes == "" {
		log.Println("nodes should not be empty.")
		flag.Usage()
		os.Exit(0)
	}

	tmp := *nodes
	if strings.Index(tmp, "|") == -1 {
		log.Printf("nodes to call: %q\n", tmp)

		dests := genServiceName(tmp)
		log.Println(dests)

		for _, d := range dests {
			go call(d, ch, "")
		}
	} else {
		after := strings.SplitN(tmp, "|", 2)
		log.Printf("nodes to call: %q\n", after[0])
		log.Printf("nodes to deliver: %q\n", after[1])

		dests := genServiceName(after[0])
		log.Println("---> genServiceName:", dests)
		for _, d := range dests {
			go call(d, ch, after[1])
		}
	}
	select {}
}

func call(addr string, ch *ocgrpc.ClientHandler, left string) {
	conn, err := grpc.Dial(addr, grpc.WithStatsHandler(ch), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	r, err := c.SayHello(context.Background(), &pb.HelloRequest{
		Name:      defaultName,
		Nodes:     left,
		ErrorRate: float32(0), // TODO
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
