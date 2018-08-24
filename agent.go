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
	"go.opencensus.io/stats/view"
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

	e.logger.Printf("[%s] SpanContext.TraceID: %s\n", sd.Name, sd.SpanContext.TraceID.String())
	e.logger.Printf("[%s] SpanContext.SpanID: %s\n", sd.Name, sd.SpanContext.SpanID.String())

	s := &traceproto.Span{
		TraceId: sd.SpanContext.TraceID[:],
		SpanId:  sd.SpanContext.SpanID[:],
		Name: &traceproto.TruncatableString{
			Value: sd.Name,
		},
		Kind: spanKind(sd),
		StartTime: &timestamp.Timestamp{
			Seconds: sd.StartTime.Unix(),
			Nanos:   int32(sd.StartTime.Nanosecond()),
		},
		EndTime: &timestamp.Timestamp{
			Seconds: sd.EndTime.Unix(),
			Nanos:   int32(sd.EndTime.Nanosecond()),
		},
		Attributes: convertToAttributes(sd.Attributes),
		//StackTrace: &traceproto.StackTrace{},
		TimeEvents: convertToTimeEvents(sd.Annotations, sd.MessageEvents),
		//Links:      &traceproto.Span_Links{},
		Status: &traceproto.Status{
			Code:    sd.Code,
			Message: sd.Message,
		},
	}

	if sd.ParentSpanID != (trace.SpanID{}) {
		s.ParentSpanId = make([]byte, 8)
		copy(s.ParentSpanId, sd.ParentSpanID[:])
		e.logger.Printf("[%s] s.ParentSpanId: %s   sd.ParentSpanID: %s\n", sd.Name, fmt.Sprintf("%02x", s.ParentSpanId[:]), sd.ParentSpanID.String())
	}

	//e.logger.Printf("[%s] spankind: %s\n", sd.Name, s.GetKind().String())

	// NOTE: do batch procession here, if need

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

// ExportView logs the view data.
func (e *Exporter) ExportView(vd *view.Data) {
	log.Println("---> ExportView:", vd)
}

func (e *Exporter) Stop() {
	e.lock.Lock()
	e.clientConn.Close()
	e.exportClient = nil
	e.lock.Unlock()
}

func convertToAttributes(tags map[string]interface{}) *traceproto.Span_Attributes {
	attributes := &traceproto.Span_Attributes{
		AttributeMap: make(map[string]*traceproto.AttributeValue),
	}

	for k, i := range tags {
		switch v := i.(type) {
		case string:
			attributes.AttributeMap[k] = &traceproto.AttributeValue{
				Value: &traceproto.AttributeValue_StringValue{
					StringValue: &traceproto.TruncatableString{
						Value: v,
					},
				},
			}
		case bool:
			attributes.AttributeMap[k] = &traceproto.AttributeValue{
				Value: &traceproto.AttributeValue_BoolValue{
					BoolValue: v,
				},
			}
		case int64:
			attributes.AttributeMap[k] = &traceproto.AttributeValue{
				Value: &traceproto.AttributeValue_IntValue{
					IntValue: int64(v),
				},
			}
		default:
			fmt.Printf("unknown tag value type:%v, ignored\n", v)
		}
	}

	return attributes
}

func convertToTimeEvents(as []trace.Annotation, ms []trace.MessageEvent) *traceproto.Span_TimeEvents {
	timeEvents := &traceproto.Span_TimeEvents{
		TimeEvent: make([]*traceproto.Span_TimeEvent, 0, 10),
	}
	for _, a := range as {
		timeEvents.TimeEvent = append(timeEvents.TimeEvent,
			&traceproto.Span_TimeEvent{
				Time:  &timestamp.Timestamp{Seconds: a.Time.Unix(), Nanos: int32(a.Time.UnixNano())},
				Value: convertAnnoationToTimeEvent(a.Attributes),
			},
		)
	}
	for _, m := range ms {
		timeEvents.TimeEvent = append(timeEvents.TimeEvent,
			&traceproto.Span_TimeEvent{
				Time:  &timestamp.Timestamp{Seconds: m.Time.Unix(), Nanos: int32(m.Time.UnixNano())},
				Value: convertMessageEventToTimeEvent(&m),
			},
		)
	}
	return timeEvents
}

func convertAnnoationToTimeEvent(annotations map[string]interface{}) *traceproto.Span_TimeEvent_Annotation_ {
	teAnnotation := &traceproto.Span_TimeEvent_Annotation_{
		Annotation: &traceproto.Span_TimeEvent_Annotation{
			Description: &traceproto.TruncatableString{
				Value: "user supplied log",
			},
			Attributes: convertToAttributes(annotations),
		},
	}
	return teAnnotation
}
func convertMessageEventToTimeEvent(m *trace.MessageEvent) *traceproto.Span_TimeEvent_MessageEvent_ {
	return nil
}

func spanKind(s *trace.SpanData) traceproto.Span_SpanKind {
	switch s.SpanKind {
	case trace.SpanKindClient:
		return trace.SpanKindClient
	case trace.SpanKindServer:
		return trace.SpanKindServer
	}
	return trace.SpanKindUnspecified
}
