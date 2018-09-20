// Package agent contains an trace exporter for Hunter.
package agent // import "github.com/moooofly/opencensus-go-exporter-hunter"

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"google.golang.org/api/support/bundler"
	"google.golang.org/grpc"

	"github.com/census-instrumentation/opencensus-proto/gen-go/exporterproto"
	"github.com/census-instrumentation/opencensus-proto/gen-go/traceproto"
)

var _ trace.Exporter = (*Exporter)(nil)

// Exporter is an implementation of trace.Exporter that export spans to Hunter agent.
type Exporter struct {
	*options
	overflowLogger

	mu           sync.Mutex
	started      bool
	stopped      bool
	clientConn   *grpc.ClientConn
	exportClient exporterproto.Export_ExportSpanClient

	bundler *bundler.Bundler
}

// NewExporter returns an implementation of trace.Exporter that exports spans
// to Hunter agent.
func NewExporter(opt ...ExporterOption) (*Exporter, error) {

	e := &Exporter{}

	opts := defaultExporterOptions
	for _, o := range opt {
		o(&opts)
	}

	preferred, err := preferedAddr(&opts)
	if err != nil {
		return nil, err
	}

	bundler := bundler.NewBundler((*trace.SpanData)(nil), func(bundle interface{}) {
		e.uploadSpans(bundle.([]*trace.SpanData))
	})

	// FIXME(moooofly): need to optimize
	// NOTE: The bundle settings here are related to memory
	if opts.bundleDelayThreshold > 0 {
		bundler.DelayThreshold = opts.bundleDelayThreshold
	} else {
		bundler.DelayThreshold = 2 * time.Second
	}

	// Once a bundle has this many items, handle the bundle. Since only one
	// item at a time is added to a bundle, no bundle will exceed this
	// threshold, so it also serves as a limit. The default is
	// DefaultBundleCountThreshold (10).
	if opts.bundleCountThreshold > 0 {
		bundler.BundleCountThreshold = opts.bundleCountThreshold
	} else {
		bundler.BundleCountThreshold = 300
	}

	// Once the number of bytes in current bundle reaches this threshold, handle
	// the bundle. The default is DefaultBundleByteThreshold (1M). This triggers handling,
	// but does not cap the total size of a bundle.
	bundler.BundleByteThreshold = bundler.BundleCountThreshold * 1000
	// The maximum size of a bundle, in bytes. Zero means unlimited.
	bundler.BundleByteLimit = bundler.BundleCountThreshold * 1000
	// The maximum number of bytes that the Bundler will keep in memory before
	// returning ErrOverflow. The default is DefaultBufferedByteLimit (1G).
	bundler.BufferedByteLimit = bundler.BundleCountThreshold * 1000 * 1000

	e.options = &opts
	e.bundler = bundler

	err = e.Start(preferred)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Exporter) doStart(proto string) error {
	if e.started {
		return nil
	}

	dialOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithTimeout(3 * time.Second),
		grpc.WithDialer(
			func(addr string, timeout time.Duration) (net.Conn, error) {
				return net.DialTimeout(proto, addr, timeout)
			}),
	}

	var cc *grpc.ClientConn
	// NOTE: In the worst case of (no agent actually available), it will take at least:
	//      (5 * 1s) + ((1<<5)-1) * 0.05 s = 5s + 1.55s = 6.55s
	dialBackoffWaitPeriod := 50 * time.Millisecond
	err := retryWithExponentialBackoff(5, dialBackoffWaitPeriod, func() error {
		var err error
		// NOTE: THIS IS A BLOCK CALL
		cc, err = grpc.Dial(e.addrs[proto], dialOpts...)
		return err
	})
	if err != nil {
		return err
	}
	e.clientConn = cc

	exportClientStream, err := exporterproto.NewExportClient(cc).ExportSpan(context.Background())
	if err != nil {
		return err
	}
	e.exportClient = exportClientStream

	return nil
}

// Start dials to the Hunter agent, establishing a connection to it.
//
// It performs a best case attempt to dial to the agent.
// It retries dialing when fail happens with:
//  * gRPC dialTimeout of 1s
//  * exponential backoff, 5 times with a period of 50ms
// hence in the worst case of (no agent actually available), it will take at least:
//      (5 * 1s) + ((1<<5)-1) * 0.05 s = 5s + 1.55s = 6.55s
func (e *Exporter) Start(proto string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	err := e.doStart(proto)
	if err == nil {
		e.started = true
		return nil
	}

	e.started = false
	if e.clientConn != nil {
		e.clientConn.Close()
	}

	return err
}

// Stop shuts down the connection and resources related to the exporter.
func (e *Exporter) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.started {
		return errors.New("not started")
	}

	if e.stopped {
		return nil
	}

	e.Flush()

	var err error
	if e.clientConn != nil {
		err = e.clientConn.Close()
	}

	e.started = false
	e.stopped = true

	return err
}

// uploadSpans uploads a set of spans
func (e *Exporter) uploadSpans(spans []*trace.SpanData) {
	if len(spans) == 0 {
		return
	}

	req := &exporterproto.ExportSpanRequest{
		Spans: make([]*traceproto.Span, 0, len(spans)),
	}

	for _, span := range spans {
		if span != nil {
			req.Spans = append(req.Spans, toProtoSpan(span))
		}
	}

	if err := e.exportClient.Send(req); err != nil {
		if err == io.EOF {
			e.logger.Println("Connection is unavailable, LOST current Span...")
		} else {
			e.onError(err)
		}
	}
}

// ExportSpan exports spans to Hunter agent.
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
		go e.uploadSpans([]*trace.SpanData{s})
	case bundler.ErrOverflow:
		e.overflowLogger.log()
	default:
		e.onError(err)
	}
}

// ExportView exports the view data.
func (e *Exporter) ExportView(vd *view.Data) {
	log.Println("---> ExportView:", vd)
}

func (e *Exporter) onError(err error) {
	if e.onError != nil {
		e.onError(err)
		return
	}
	log.Printf("Fail: %v", err)
}

// Flush waits for exported trace spans to be uploaded.
//
// This is useful if your program is ending and you do not want to lose recent
// spans.
func (e *Exporter) Flush() {
	e.bundler.Flush()
}
