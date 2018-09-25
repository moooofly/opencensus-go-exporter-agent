package wrapper

import (
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc/stats"
)

type ServerExtHandler struct {
	sh *ocgrpc.ServerHandler
	si *ServerCustomInfo
}

var _ stats.Handler = (*ServerExtHandler)(nil)

func NewServerExtHandler(in *ServerCustomInfo) *ServerExtHandler {
	return &ServerExtHandler{
		si: in,
		sh: &ocgrpc.ServerHandler{},
	}
}

// HandleConn exists to satisfy gRPC stats.Handler.
func (s *ServerExtHandler) HandleConn(ctx context.Context, cs stats.ConnStats) {
	// no-op
	s.sh.HandleConn(ctx, cs)
}

// TagConn exists to satisfy gRPC stats.Handler.
func (s *ServerExtHandler) TagConn(ctx context.Context, cti *stats.ConnTagInfo) context.Context {
	// no-op
	return s.sh.TagConn(ctx, cti)
}

// HandleRPC implements per-RPC tracing and stats instrumentation.
func (s *ServerExtHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	// do something here
	s.sh.HandleRPC(ctx, rs)
}

// TagRPC implements per-RPC context management.
func (s *ServerExtHandler) TagRPC(ctx context.Context, rti *stats.RPCTagInfo) context.Context {
	ctx = s.sh.TagRPC(ctx, rti)

	span := trace.FromContext(ctx)
	serverInfoAdd(span, s.si)

	return ctx
}

func serverInfoAdd(span *trace.Span, info *ServerCustomInfo) {
	span.AddAttributes(trace.StringAttribute("service_name", info.ServiceName))
	span.AddAttributes(trace.StringAttribute("hostname", info.HostName))

	span.AddAttributes(trace.StringAttribute("kind", info.Kind))
}
