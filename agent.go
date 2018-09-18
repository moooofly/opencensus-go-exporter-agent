// Package agent contains an trace exporter for Hunter.
package agent // import "github.com/moooofly/opencensus-go-exporter-agent"

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"google.golang.org/api/support/bundler"
	"google.golang.org/grpc"

	"github.com/census-instrumentation/opencensus-proto/gen-go/exporterproto"
	"github.com/census-instrumentation/opencensus-proto/gen-go/traceproto"
)

const DefaultConfigPath = "/etc/podinfo/labels"

const DefaultUnixSocketEndpoint = "/var/run/hunter-agent.sock"
const DefaultTCPPort = 12345
const DefaultTCPHost = "0.0.0.0"

var DefaultTCPEndpoint = fmt.Sprintf("%s:%d", DefaultTCPHost, DefaultTCPPort)

// Exporter is an implementation of trace.Exporter that export spans to Hunter agent.
type Exporter struct {
	*options

	bundler *bundler.Bundler
	// uploadFn defaults to uploadSpans;
	uploadFn func(spans []*trace.SpanData)

	overflowLogger

	lock         sync.Mutex
	clientConn   *grpc.ClientConn
	exportClient exporterproto.Export_ExportSpanClient
}

var _ trace.Exporter = (*Exporter)(nil)

// options are the options to be used when initializing the Hunter agent exporter.
type options struct {
	// Hunter agent listening address
	addrs  map[string]string
	logger *log.Logger

	// OnError is the hook to be called when there is
	// an error occurred.
	// If no custom hook is set, errors are logged.
	// Optional.
	onError func(err error)

	// BundleDelayThreshold determines the max amount of time
	// the exporter can wait before uploading data to the backend.
	// Optional.
	BundleDelayThreshold time.Duration
	// BundleCountThreshold determines how many data events
	// can be buffered before batch uploading them to the backend.
	// Optional.
	BundleCountThreshold int
}

var defaultExporterOptions = options{
	addrs: map[string]string{
		"tcp": DefaultTCPEndpoint,
		//"unix": DefaultUnixSocketEndpoint,
	},
	logger:               log.New(os.Stderr, "[hunter-agent-exporter] ", log.LstdFlags),
	onError:              nil,
	BundleDelayThreshold: 2 * time.Second,
	BundleCountThreshold: 50,
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

	bundler := bundler.NewBundler((*trace.SpanData)(nil), func(bundle interface{}) {
		e.uploadFn(bundle.([]*trace.SpanData))
	})

	if e.BundleDelayThreshold > 0 {
		bundler.DelayThreshold = e.BundleDelayThreshold
	} else {
		bundler.DelayThreshold = 2 * time.Second
	}

	if e.BundleCountThreshold > 0 {
		bundler.BundleCountThreshold = e.BundleCountThreshold
	} else {
		bundler.BundleCountThreshold = 50
	}

	bundler.BundleByteThreshold = bundler.BundleCountThreshold * 200
	bundler.BundleByteLimit = bundler.BundleCountThreshold * 1000
	bundler.BufferedByteLimit = bundler.BundleCountThreshold * 2000

	e.bundler = bundler
	e.uploadFn = e.uploadSpans

	return e, nil
}

func (e *Exporter) onError(err error) {
	if e.onError != nil {
		e.onError(err)
		return
	}
	e.logger.Printf("Exporter fail: %v", err)
}

// uploadSpans uploads a set of spans
func (e *Exporter) uploadSpans(spans []*trace.SpanData) {
	e.lock.Lock()
	exportClient := e.exportClient
	e.lock.Unlock()

	if exportClient == nil {
		return
	}

	req := &exporterproto.ExportSpanRequest{
		Spans: make([]*traceproto.Span, 0, len(spans)),
	}

	for _, span := range spans {
		req.Spans = append(req.Spans, protoFromSpanData(span))
	}

	if err := exportClient.Send(req); err != nil {
		if err == io.EOF {
			e.logger.Println("Connection is unavailable, LOST current Span...")
			e.Stop()
		} else {
			e.onError(err)
		}
	}
}

func protoFromSpanData(s *trace.SpanData) *traceproto.Span {
	//e.logger.Printf("[%s] SpanContext.TraceID: %s\n", s.Name, s.SpanContext.TraceID.String())
	//e.logger.Printf("[%s] SpanContext.SpanID: %s\n", s.Name, s.SpanContext.SpanID.String())
	if s == nil {
		return nil
	}

	sp := &traceproto.Span{
		TraceId: s.SpanContext.TraceID[:],
		SpanId:  s.SpanContext.SpanID[:],
		Name: &traceproto.TruncatableString{
			Value: s.Name,
		},
		Kind: spanKind(s),
		StartTime: &timestamp.Timestamp{
			Seconds: s.StartTime.Unix(),
			Nanos:   int32(s.StartTime.Nanosecond()),
		},
		EndTime: &timestamp.Timestamp{
			Seconds: s.EndTime.Unix(),
			Nanos:   int32(s.EndTime.Nanosecond()),
		},
		Attributes: convertToAttributes(s.Attributes),
		//StackTrace: &traceproto.StackTrace{},
		TimeEvents: convertToTimeEvents(s.Annotations, s.MessageEvents),
		//Links:      &traceproto.Span_Links{},
		Status: &traceproto.Status{
			Code:    s.Code,
			Message: s.Message,
		},
	}

	if s.ParentSpanID != (trace.SpanID{}) {
		sp.ParentSpanId = make([]byte, 8)
		copy(sp.ParentSpanId, s.ParentSpanID[:])
		//e.logger.Printf("[%s] s.ParentSpanId: %s   s.ParentSpanID: %s\n", s.Name, fmt.Sprintf("%02x", s.ParentSpanId[:]), s.ParentSpanID.String())
	}

	return sp
}

// ExportSpan exports a span to Hunter agent.
func (e *Exporter) ExportSpan(s *trace.SpanData) {
	n := 1
	n += len(s.Attributes)
	n += len(s.Annotations)
	n += len(s.MessageEvents)
	err := e.bundler.Add(s, n)
	switch err {
	case nil:
		return
	case bundler.ErrOversizedItem:
		go e.uploadFn([]*trace.SpanData{s})
	case bundler.ErrOverflow:
		e.overflowLogger.log()
	default:
		e.onError(err)
	}
}

// Flush waits for exported trace spans to be uploaded.
//
// This is useful if your program is ending and you do not want to lose recent
// spans.
func (e *Exporter) Flush() {
	e.bundler.Flush()
}

// overflowLogger ensures that at most one overflow error log message is
// written every 5 seconds.
type overflowLogger struct {
	mu    sync.Mutex
	pause bool
	accum int
}

func (o *overflowLogger) delay() {
	o.pause = true
	time.AfterFunc(5*time.Second, func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		switch {
		case o.accum == 0:
			o.pause = false
		case o.accum == 1:
			log.Println("OpenCensus agent exporter: failed to upload span: buffer full")
			o.accum = 0
			o.delay()
		default:
			log.Printf("OpenCensus agent exporter: failed to upload %d spans: buffer full", o.accum)
			o.accum = 0
			o.delay()
		}
	})
}

func (o *overflowLogger) log() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.pause {
		log.Println("OpenCensus agent exporter: failed to upload span: buffer full")
		o.delay()
	} else {
		o.accum++
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

// configRead reads value by specific config key
func ConfigRead(path string, key string) string {
	if path == "" {
		path = DefaultConfigPath
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println("--> find no file: ", path)
		return ""
	}

	content := string(b)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		kv := strings.Split(line, "=")
		if kv[0] == key {
			log.Printf("--> match key[%s], value is: %s\n", key, kv[1])
			return strings.Trim(kv[1], "\"")
		}
	}
	return ""
}
