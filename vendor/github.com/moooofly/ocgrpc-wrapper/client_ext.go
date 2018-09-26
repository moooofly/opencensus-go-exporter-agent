package wrapper

import (
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc/stats"
)

type ClientExtHandler struct {
	ch *ocgrpc.ClientHandler
	ci *ClientCustomInfo
}

func NewClientExtHandler(in *ClientCustomInfo) *ClientExtHandler {
	return &ClientExtHandler{
		ci: in,
		ch: &ocgrpc.ClientHandler{
			StartOptions: trace.StartOptions{
				Sampler: in.Sampler,
			},
		},
	}
}

// HandleConn exists to satisfy gRPC stats.Handler.
func (c *ClientExtHandler) HandleConn(ctx context.Context, cs stats.ConnStats) {
	// no-op
	c.ch.HandleConn(ctx, cs)
}

// TagConn exists to satisfy gRPC stats.Handler.
func (c *ClientExtHandler) TagConn(ctx context.Context, cti *stats.ConnTagInfo) context.Context {
	// no-op
	return c.ch.TagConn(ctx, cti)
}

// HandleRPC implements per-RPC tracing and stats instrumentation.
func (c *ClientExtHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	c.ch.HandleRPC(ctx, rs)
}

// TagRPC implements per-RPC context management.
func (c *ClientExtHandler) TagRPC(ctx context.Context, rti *stats.RPCTagInfo) context.Context {
	ctx = c.ch.TagRPC(ctx, rti)

	span := trace.FromContext(ctx)
	clientInfoAdd(span, c.ci)

	return ctx
}

func clientInfoAdd(span *trace.Span, info *ClientCustomInfo) {
	span.AddAttributes(trace.StringAttribute("service_name", info.ServiceName))
	span.AddAttributes(trace.StringAttribute("hostname", info.HostName))

	span.AddAttributes(trace.StringAttribute("remote_kind", info.RemoteKind))
}
