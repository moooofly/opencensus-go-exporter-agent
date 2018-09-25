package agent

import (
	"fmt"

	"github.com/census-instrumentation/opencensus-proto/gen-go/traceproto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.opencensus.io/trace"
)

func toProtoSpan(s *trace.SpanData) *traceproto.Span {
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
	}

	return sp
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
