// Package agent contains an trace exporter for Hunter.
package agent // import "github.com/moooofly/opencensus-go-exporter-agent"

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"

	"github.com/census-instrumentation/opencensus-proto/gen-go/exporterproto"
	"github.com/census-instrumentation/opencensus-proto/gen-go/traceproto"
)

const DefaultUnixSocketEndpoint = "/var/run/hunter-agent.sock"
const DefaultTCPPort = 12345
const DefaultTCPHost = "0.0.0.0"

var DefaultTCPEndpoint = fmt.Sprintf("%s:%d", DefaultTCPHost, DefaultTCPPort)

// Exporter is an implementation of trace.Exporter that export spans to Hunter agent.
type Exporter struct {
	*options

	lock         sync.Mutex
	clientConn   *grpc.ClientConn
	exportClient exporterproto.Export_ExportSpanClient
}

var _ trace.Exporter = (*Exporter)(nil)

// options are the options to be used when initializing the Hunter agent exporter.
type options struct {
	// Hunter agent listening address
	addrs   map[string]string
	logger  *log.Logger
	onError func(err error)

	// TODO: add more options here
}

var defaultExporterOptions = options{
	addrs: map[string]string{
		"tcp": DefaultTCPEndpoint,
		//"unix": DefaultUnixSocketEndpoint,
	},
	logger:  log.New(os.Stderr, "[hunter-agent-exporter] ", log.LstdFlags),
	onError: nil,
}

// ExporterOption sets options such as addrs, logger, etc.
type ExporterOption func(*options)

// Logger sets the logger used to report errors.
func Addrs(addrs map[string]string) ExporterOption {
	return func(o *options) {
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
	if _, ok := opts.addrs["tcp"]; ok {
		preferred = "tcp"
	}
	if _, ok := opts.addrs["unix"]; ok {
		preferred = "unix"
	}

	if preferred == "" {
		return nil, errors.New("find no addrs")
	}

	log.Println("===> preferred:", opts.addrs[preferred])

	// FIXME: need to reconnect?
	conn, err := grpc.Dial(opts.addrs[preferred], grpc.WithInsecure(), grpc.WithTimeout(10*time.Second),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout(preferred, addr, timeout)
		}))
	if err != nil {
		return nil, err
	}

	clientStream, err := exporterproto.NewExportClient(conn).ExportSpan(context.Background())
	if err != nil {
		return nil, err
	}

	e := &Exporter{
		options:      &opts,
		clientConn:   conn,
		exportClient: clientStream,
	}

	return e, nil
}

func (e *Exporter) onError(err error) {
	if e.onError != nil {
		e.onError(err)
		return
	}
	e.logger.Printf("Exporter fail: %v", err)
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
	// e.logger.Printf("Current trace.SpanData: \n%#v\n", *sd)

	e.logger.Printf("[%s] SpanContext.TraceID: %s\n", sd.Name, sd.SpanContext.TraceID.String())
	e.logger.Printf("[%s] SpanContext.SpanID: %s\n", sd.Name, sd.SpanContext.SpanID.String())

	s := &traceproto.Span{
		TraceId:      sd.SpanContext.TraceID[:],
		SpanId:       sd.SpanContext.SpanID[:],
		ParentSpanId: sd.ParentSpanID[:],
		Name: &traceproto.TruncatableString{
			Value: sd.Name,
		},
		Kind: traceproto.Span_CLIENT,
		StartTime: &timestamp.Timestamp{
			Seconds: sd.StartTime.Unix(),
			Nanos:   int32(sd.StartTime.Nanosecond()),
		},
		EndTime: &timestamp.Timestamp{
			Seconds: sd.EndTime.Unix(),
			Nanos:   int32(sd.EndTime.Nanosecond()),
		},
		// TODO: Add attributes and others.
		Attributes: &traceproto.Span_Attributes{},
		StackTrace: &traceproto.StackTrace{},
		TimeEvents: &traceproto.Span_TimeEvents{},
		Links:      &traceproto.Span_Links{},
		Status:     &traceproto.Status{},
	}

	// FIXME: actually the code below send one item a time only.
	if err := exportClient.Send(&exporterproto.ExportSpanRequest{
		Spans: []*traceproto.Span{s},
	}); err != nil {
		if err == io.EOF {
			e.logger.Println("Connection is unavailable, LOST current Span...")
			e.Stop()
		} else {
			e.onError(err)
		}
	}
}

func (e *Exporter) Stop() {
	e.lock.Lock()
	e.clientConn.Close()
	e.exportClient = nil
	e.lock.Unlock()
}
