package interceptors

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tehrelt/mu-lib/tracer"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		t := otel.Tracer(tracer.TracerKey)
		ctx, span := t.Start(ctx, method)
		defer span.End()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		propagator := propagation.TraceContext{}
		carrier := propagation.MapCarrier{}
		propagator.Inject(ctx, carrier)
		for k, v := range carrier {
			md.Set(k, v)
		}

		ctx = metadata.NewOutgoingContext(ctx, md)
		payload, err := json.Marshal(req)
		if err != nil {
			return err
		}

		err = invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			span.RecordError(err)
		}

		defer func() {
			span.SetAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.method", method),
				attribute.String("rpc.peer_address", cc.Target()),
				attribute.Int64("rpc.duration_ms", time.Since(start).Milliseconds()),
				attribute.String("rpc.payload", string(payload)),
			)
		}()

		return err
	}
}

// StreamClientInterceptor returns a stream client interceptor for OpenTelemetry tracing
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		propagator := propagation.TraceContext{}
		carrier := propagation.MapCarrier{}
		propagator.Inject(ctx, carrier)
		for k, v := range carrier {
			md.Set(k, v)
		}

		ctx, span := otel.Tracer(tracer.TracerKey).Start(
			ctx,
			method,
			trace.WithSpanKind(trace.SpanKindClient),
		)

		ctx = metadata.NewOutgoingContext(ctx, md)

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return nil, err
		}

		span.SetAttributes(
			attribute.String("rpc.method", method),
			attribute.String("rpc.system", "grpc"),
			attribute.String("rpc.peer_address", cc.Target()),
		)

		return &tracingClientStream{
			ClientStream: stream,
			span:         span,
			start:        start,
		}, nil
	}
}
