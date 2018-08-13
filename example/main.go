package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	agent "github.com/moooofly/opencensus-go-exporter-agent"
	"go.opencensus.io/trace"
)

var (
	tcpAddr      = flag.String("tcp_addr", os.Getenv("AGENT_TCP_ADDR"), "The TCP endport of hunter agent.")
	unixsockAddr = flag.String("unix_sock_addr", os.Getenv("AGENT_UNIX_ADDR"), "The Unix socket endpoint of hunter agent.")
)

var logger *log.Logger = log.New(os.Stderr, "[example] ", log.LstdFlags)

func main() {
	flag.Parse()

	if *tcpAddr == "" && *unixsockAddr == "" {
		flag.Usage()
		os.Exit(0)
	}

	addrs := make([]string, 0)

	if *tcpAddr != "" {
		// check
		logger.Printf("The TCP endpoint of Hunter agent: %s", *tcpAddr)
		addrs = append(addrs, *tcpAddr)
	}
	if *unixsockAddr != "" {
		// check
		logger.Printf("The Unix socket endpoint of Hunter agent: %s", *unixsockAddr)
		addrs = append(addrs, *unixsockAddr)
	}

	exporter, err := agent.NewExporter(
		agent.Addrs(addrs),
		//agent.Logger(logger),
		//agent.Topic(*topic),
	)
	if err != nil {
		logger.Println("err:", err)
		os.Exit(1)
	}

	trace.RegisterExporter(exporter)

	// For example purposes, sample every trace.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	for {
		foo(context.Background())
		time.Sleep(time.Second)
		logger.Println("-----")
	}
}

func foo(ctx context.Context) {
	// Name the current span "/foo"
	ctx, span := trace.StartSpan(ctx, "/foo")
	logger.Println("foo ->")
	defer span.End()

	// Foo calls bar and baz
	bar(ctx)
	baz(ctx)
}

func bar(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "/bar")
	logger.Println("  bar ->")
	defer span.End()

	// Do bar
	time.Sleep(2 * time.Millisecond)
}

func baz(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "/baz")
	logger.Println("  baz ->")
	defer span.End()

	// Do baz
	time.Sleep(4 * time.Millisecond)
}
