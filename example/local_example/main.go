package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	agent "github.com/moooofly/opencensus-go-exporter-agent"
	"go.opencensus.io/trace"
)

// used as attributes key
const (
	SERVICE_NAME = "service_name"
	KIND         = "kind"
	REMOTE_KIND  = "remote_kind"
)

// used as value of REMOTE_KIND
const (
	KIND_GRPC  = "grpc"
	KIND_HTTP  = "http"
	KIND_MYSQL = "mysql"
	KIND_REDIS = "redis"
)

var (
	// NOTE: should obtain this from $HOST_IP env
	//tcpAddr = flag.String("tcp_addr", os.Getenv("AGENT_TCP_ADDR"),
	tcpAddr = flag.String("tcp_addr", os.Getenv("HOST_IP"),
		"The TCP endport of Hunter agent, can also set with AGENT_TCP_ADDR env. (Format: tcp://<host>:<port>)")
	unixsockAddr = flag.String("unix_sock_addr", os.Getenv("AGENT_UNIX_ADDR"),
		"The Unix endpoint of Hunter agent, can also set with AGENT_UNIX_ADDR env. (Format: unix:///<path-to-unix-domain>)")

	// NOTE: should obtain this from $HOSTNAME env
	hostName = flag.String("hostname", os.Getenv("HOSTNAME"), "As an Attribute of span.")

	// obtain service_name from config file
	fakeconfig  = "config.fake"
	serviceName = agent.ConfigRead(fakeconfig, "cluster")
)

var logger *log.Logger = log.New(os.Stderr, "[example] ", log.LstdFlags)

func main() {
	flag.Parse()

	if *tcpAddr == "" && *unixsockAddr == "" {
		flag.Usage()
		os.Exit(0)
	}

	addrs := make(map[string]string, 2)

	if *tcpAddr != "" {
		// NOTE: should check TCP endport format here.
		logger.Printf("The TCP endpoint of Hunter agent: %s", *tcpAddr)
		addrs["tcp"] = strings.TrimPrefix(*tcpAddr, "tcp://")
	}
	if *unixsockAddr != "" {
		// NOTE: should check Unix endport format here.
		logger.Printf("The Unix endpoint of Hunter agent: %s", *unixsockAddr)
		addrs["unix"] = strings.TrimPrefix(*unixsockAddr, "unix://")
	}

	exporter, err := agent.NewExporter(
		agent.Addrs(addrs),
		//agent.Logger(logger),
	)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}
	defer exporter.Flush()

	trace.RegisterExporter(exporter)

	// For example purposes, sample every trace.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	for {
		simulate_neo_api(context.Background())
		//time.Sleep(10 * time.Millisecond)
		logger.Println("-----")
	}
}

// NOTE: update this value every time
var index = "20180912"

func simulate_neo_api(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx,
		"/simulate_neo_api",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	logger.Println("simulate_neo_api ->")
	defer span.End()

	span.AddAttributes(trace.StringAttribute(SERVICE_NAME, index+"-"+serviceName))
	span.SetName("/api/user/:uid/profile")
	span.AddAttributes(trace.StringAttribute(KIND, KIND_HTTP))
	span.AddAttributes(trace.Int64Attribute("uid", int64(123456)))
	span.AddAttributes(trace.StringAttribute("hostname", "fake-hostname-1"))

	span.Annotate([]trace.Attribute{
		trace.StringAttribute("query", "/api/user/123456/profile?from=web&version=1.0.1..."),
	}, "Annotate")

	span.SetStatus(trace.Status{Code: int32(0), Message: "ok"})

	simulate_grpc_client(ctx)
	simulate_neo_api_call_mysql(ctx)
	simulate_neo_api_call_redis(ctx)
}

func simulate_grpc_client(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx,
		"/simulate_grpc_client",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	logger.Println("  simulate_grpc_client ->")
	defer span.End()

	span.AddAttributes(trace.StringAttribute(SERVICE_NAME, index+"-"+serviceName))
	span.SetName("GetUserProfile")
	span.AddAttributes(trace.StringAttribute(REMOTE_KIND, KIND_GRPC))
	span.AddAttributes(trace.Int64Attribute("uid", int64(123456)))
	span.AddAttributes(trace.StringAttribute("source", "web"))
	span.AddAttributes(trace.StringAttribute("hostname", "fake-hostname-1"))

	span.SetStatus(trace.Status{Code: int32(4), Message: "DeadlineExceeded"})

	//time.Sleep(2 * time.Millisecond)
	simulate_grpc_server(ctx)
}

func simulate_grpc_server(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx,
		"/simulate_grpc_server",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	logger.Println("    simulate_grpc_server ->")
	defer span.End()

	span.AddAttributes(trace.StringAttribute(SERVICE_NAME, index+"="+serviceName))
	span.SetName("GetUserProfile")
	span.AddAttributes(trace.StringAttribute(KIND, KIND_GRPC))
	span.AddAttributes(trace.Int64Attribute("uid", int64(123456)))
	span.AddAttributes(trace.StringAttribute("source", "web"))
	span.AddAttributes(trace.StringAttribute("hostname", "fake-hostname-2"))

	span.SetStatus(trace.Status{Code: int32(0), Message: "ok"})

	//time.Sleep(4 * time.Millisecond)
	simulate_grpc_server_call_mysql(ctx)
}

func simulate_grpc_server_call_mysql(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx,
		"/simulate_grpc_server_call_mysql",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	logger.Println("    simulate_grpc_server_call_mysql ->")
	defer span.End()

	span.AddAttributes(trace.StringAttribute(SERVICE_NAME, index+"="+serviceName))
	span.SetName("select")
	span.AddAttributes(trace.StringAttribute(REMOTE_KIND, KIND_MYSQL))
	span.AddAttributes(trace.Int64Attribute("uid", int64(123456)))
	span.AddAttributes(trace.StringAttribute("source", "grpc"))
	span.AddAttributes(trace.StringAttribute("hostname", "fake-hostname-2"))

	span.Annotate([]trace.Attribute{
		trace.StringAttribute("query", "select * from user where uid=123456"),
	}, "Annotate")

	span.SetStatus(trace.Status{Code: int32(0), Message: "ok"})

	//time.Sleep(15 * time.Millisecond)
}

func simulate_neo_api_call_mysql(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx,
		"/simulate_neo_api_call_mysql",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	logger.Println("  simulate_neo_api_call_mysql ->")
	defer span.End()

	span.AddAttributes(trace.StringAttribute(SERVICE_NAME, index+"-"+serviceName))
	span.SetName("select")
	span.AddAttributes(trace.StringAttribute(REMOTE_KIND, KIND_MYSQL))
	span.AddAttributes(trace.Int64Attribute("uid", int64(123456)))
	span.AddAttributes(trace.StringAttribute("source", "web"))
	span.AddAttributes(trace.StringAttribute("hostname", "fake-hostname-1"))

	span.Annotate([]trace.Attribute{
		trace.StringAttribute("query", "select * from profile where uid=123456"),
	}, "Annotate")

	span.SetStatus(trace.Status{Code: int32(0), Message: "ok"})

	//time.Sleep(25 * time.Millisecond)
}

func simulate_neo_api_call_redis(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx,
		"/simulate_neo_api_call_redis",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	logger.Println("  simulate_neo_api_call_redis ->")
	defer span.End()

	span.AddAttributes(trace.StringAttribute(SERVICE_NAME, index+"-"+serviceName))
	span.SetName("mget")
	span.AddAttributes(trace.StringAttribute(REMOTE_KIND, KIND_REDIS))
	span.AddAttributes(trace.Int64Attribute("uid", int64(123456)))
	span.AddAttributes(trace.StringAttribute("source", "web"))
	span.AddAttributes(trace.Int64Attribute("count", int64(1000)))
	span.AddAttributes(trace.StringAttribute("hostname", "fake-hostname-1"))

	span.Annotate([]trace.Attribute{
		trace.StringAttribute("query", "mget 1,2,3,4..."),
	}, "Annotate")

	span.SetStatus(trace.Status{Code: int32(0), Message: "ok"})

	//time.Sleep(35 * time.Millisecond)
}
