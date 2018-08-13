// Package agent contains an trace exporter for Hunter.
package agent // import "github.com/moooofly/opencensus-go-exporter-agent"

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/golang/protobuf/ptypes/timestamp"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"

	"github.com/census-instrumentation/opencensus-proto/gen-go/traceproto"
	"github.com/moooofly/opencensus-go-exporter-agent/gen-go/dumpproto"
)

var DefaultTCPPort = 12345
var DefaultTCPHost = "localhost"
var DefaultTCPEndpoint = fmt.Sprintf("tcp://%s:%d", DefaultTCPHost, DefaultTCPPort)
var DefaultUnixSocketEndpoint = "/var/run/hunter-agent.sock"

// Exporter is an implementation of trace.Exporter that dump spans to Hunter agent.
type Exporter struct {
	opts options

	lock       sync.Mutex
	dumpClient dumpproto.Dump_ExportSpanClient
}

var _ trace.Exporter = (*Exporter)(nil)

// options are the options to be used when initializing the hunter agent exporter.
type options struct {
	// Hunter agent addresses
	addrs   []string
	logger  *log.Logger
	onError func(err error)

	// TODO: add more options here
}

var defaultExporterOptions = options{
	addrs:   []string{},
	logger:  log.New(os.Stderr, "[hunter-agent-exporter] ", log.LstdFlags),
	onError: nil,
}

// ExporterOption sets options such as addrs, logger, etc.
type ExporterOption func(*options)

// Logger sets the logger used to report errors.
func Addrs(addrs []string) ExporterOption {
	return func(o *options) {
		o.addrs = addrs
	}
}

// Logger sets the logger used to report errors.
func Logger(logger *log.Logger) ExporterOption {
	return func(o *options) {
		o.logger = logger
	}
}

//
func ErrFun(errFun func(err error)) ExporterOption {
	return func(o *options) {
		o.onError = errFun
	}
}

// NewExporter returns an implementation of trace.Exporter that dumps spans
// to Hunter agent.
func NewExporter(opt ...ExporterOption) (*Exporter, error) {

	opts := defaultExporterOptions
	for _, o := range opt {
		o(&opts)
	}

	// Addrs should be :
	// 1. TCP socket endpoint: tcp://<host>:<port>
	// 2. Unix domain sockert endpoint: unix://</path/to/unix/sock>

	// FIXME:
	conn, err := grpc.Dial(DefaultUnixSocketEndpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	//client := dumpproto.NewDumpClient(conn)
	clientStream, err := dumpproto.NewDumpClient(conn).ExportSpan(context.Background())
	if err != nil {
		return nil, err
	}

	e := &Exporter{
		opts:       opts,
		dumpClient: clientStream,
	}

	return e, nil
}

func (e *Exporter) onError(err error) {
	if e.opts.onError != nil {
		e.opts.onError(err)
		return
	}
	log.Printf("Exporter fail: %v", err)
}

// ExportSpan exports a span to hunter agent.
func (e *Exporter) ExportSpan(sd *trace.SpanData) {

	e.lock.Lock()
	dumpClient := e.dumpClient
	e.lock.Unlock()

	if dumpClient == nil {
		return
	}

	// NOTE: do batch procession here, if need

	// NOTE: the code below outputs too much
	// e.opts.logger.Printf("Current trace.SpanData: \n%#v\n", *sd)

	e.opts.logger.Printf("[%s] SpanContext.TraceID: %s\n", sd.Name, sd.SpanContext.TraceID.String())
	e.opts.logger.Printf("[%s] SpanContext.SpanID: %s\n", sd.Name, sd.SpanContext.SpanID.String())

	s := &traceproto.Span{
		TraceId:      sd.SpanContext.TraceID[:],
		SpanId:       sd.SpanContext.SpanID[:],
		ParentSpanId: sd.ParentSpanID[:],
		Name: &traceproto.TruncatableString{
			Value: sd.Name,
		},
		StartTime: &timestamp.Timestamp{
			Seconds: sd.StartTime.Unix(),
			Nanos:   int32(sd.StartTime.Nanosecond()),
		},
		EndTime: &timestamp.Timestamp{
			Seconds: sd.EndTime.Unix(),
			Nanos:   int32(sd.EndTime.Nanosecond()),
		},
		// TODO: Add attributes and others.
	}

	// FIXME: actually the code below send one item a time only.
	if err := dumpClient.Send(&dumpproto.DumpSpanRequest{
		Spans: []*traceproto.Span{s},
	}); err != nil {
		if err == io.EOF {
			e.opts.logger.Println("Connection is unavailable; will try to reconnect in a minute")
			e.deleteClient()
		} else {
			e.opts.onError(err)
		}
	}
}

func (e *Exporter) deleteClient() {
	e.lock.Lock()
	e.dumpClient = nil
	e.lock.Unlock()
}
