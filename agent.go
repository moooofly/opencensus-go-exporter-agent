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

	"github.com/census-instrumentation/opencensus-proto/gen-go/exporterproto"
	"github.com/census-instrumentation/opencensus-proto/gen-go/traceproto"
)

var DefaultTCPPort = 12345
var DefaultTCPHost = "0.0.0.0"
var DefaultTCPEndpoint = fmt.Sprintf("%s:%d", DefaultTCPHost, DefaultTCPPort)
var DefaultUnixSocketEndpoint = "/var/run/hunter-agent.sock"

// Exporter is an implementation of trace.Exporter that export spans to Hunter agent.
type Exporter struct {
	opts options

	lock         sync.Mutex
	exportClient exporterproto.Export_ExportSpanClient
}

var _ trace.Exporter = (*Exporter)(nil)

// options are the options to be used when initializing the Hunter agent exporter.
type options struct {
	// Hunter agent addresses
	addrs   map[string]string
	logger  *log.Logger
	onError func(err error)

	// TODO: add more options here
}

var defaultExporterOptions = options{
	//addrs:   make(map[string]string, 2),
	addrs:   map[string]string{"tcp": DefaultTCPEndpoint},
	logger:  log.New(os.Stderr, "[hunter-agent-exporter] ", log.LstdFlags),
	onError: nil,
}

// ExporterOption sets options such as addrs, logger, etc.
type ExporterOption func(*options)

// Logger sets the logger used to report errors.
func Addrs(addrs map[string]string) ExporterOption {
	return func(o *options) {
		if len(addrs) == 0 {
			fmt.Printf("===> Use DefaultTCPEndpoint (%s)\n", DefaultTCPEndpoint)
		}
		for k, v := range addrs {
			o.addrs[k] = v
		}
	}
}

// Logger sets the logger used to report errors.
func Logger(logger *log.Logger) ExporterOption {
	return func(o *options) {
		o.logger = logger
	}
}

// ErrFun sets xxx
func ErrFun(errFun func(err error)) ExporterOption {
	return func(o *options) {
		o.onError = errFun
	}
}

// NewExporter returns an implementation of trace.Exporter that exports spans
// to Hunter agent.
func NewExporter(opt ...ExporterOption) (*Exporter, error) {

	opts := defaultExporterOptions
	for _, o := range opt {
		o(&opts)
	}

	// NOTE: unix domain socket is preferred
	var preferred string
	if tcpAddr, ok := opts.addrs["tcp"]; ok {
		preferred = tcpAddr
	}
	if unixAddr, ok := opts.addrs["unix"]; ok {
		preferred = unixAddr
	}

	fmt.Println("===> preferred:", preferred)

	// FIXME: need to reconnect?
	conn, err := grpc.Dial(preferred, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	clientStream, err := exporterproto.NewExportClient(conn).ExportSpan(context.Background())
	if err != nil {
		return nil, err
	}

	e := &Exporter{
		opts:         opts,
		exportClient: clientStream,
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

// ExportSpan exports a span to Hunter agent.
func (e *Exporter) ExportSpan(sd *trace.SpanData) {

	e.lock.Lock()
	exportClient := e.exportClient
	e.lock.Unlock()

	if exportClient == nil {
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
	if err := exportClient.Send(&exporterproto.ExportSpanRequest{
		Spans: []*traceproto.Span{s},
	}); err != nil {
		if err == io.EOF {
			e.opts.logger.Println("Connection is unavailable, LOST current Span...")
			e.deleteClient()
		} else {
			e.opts.onError(err)
		}
	}
}

func (e *Exporter) deleteClient() {
	e.lock.Lock()
	e.exportClient = nil
	e.lock.Unlock()
}
