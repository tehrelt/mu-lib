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
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		startTime := time.Now()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		propagator := propagation.TraceContext{}
		carrier := propagation.MapCarrier{}
		for k, v := range md {
			if len(v) > 0 {
				carrier[k] = v[0]
			}
		}

		ctx = propagator.Extract(ctx, carrier)

		payload, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}

		t := otel.Tracer(tracer.TracerKey)
		ctx, span := t.Start(
			ctx,
			info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.service", info.FullMethod),
				attribute.String("rpc.payload", string(payload)),
			),
		)
		defer span.End()

		resp, err = handler(ctx, req)
		if err != nil {
			if e, ok := status.FromError(err); ok {
				span.SetStatus(codes.Error, e.Message())
			} else {
				span.SetStatus(codes.Error, err.Error())
			}

			span.RecordError(err)
		}

		defer func() {
			span.SetAttributes(
				attribute.Int64("rpc.duration_ms", time.Since(startTime).Milliseconds()),
			)

			respjson, err := json.Marshal(resp)
			if err != nil {
				return
			}

			span.SetAttributes(
				attribute.String("rpc.response", string(respjson)),
			)
		}()

		return resp, err
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		startTime := time.Now()

		// Extract trace context from incoming metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Extract the span context from the metadata
		propagator := propagation.TraceContext{}
		carrier := propagation.MapCarrier{}
		for k, v := range md {
			if len(v) > 0 {
				carrier[k] = v[0]
			}
		}

		ctx = propagator.Extract(ctx, carrier)

		// Start a new span
		t := otel.Tracer(tracer.TracerKey)
		ctx, span := t.Start(
			ctx,
			info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", info.FullMethod),
			),
		)
		defer span.End()

		// Wrap the server stream with our context
		wrappedStream := &tracedServerStream{
			ServerStream: ss,
			ctx:          ctx,
			span:         span,
		}

		err := handler(srv, wrappedStream)

		if err != nil {
			if e, ok := status.FromError(err); ok {
				span.SetStatus(codes.Error, e.Message())
			} else {
				span.SetStatus(codes.Error, err.Error())
			}

			span.RecordError(err)
		}

		span.SetAttributes(
			attribute.Int64("rpc.duration_ms", time.Since(startTime).Milliseconds()),
		)

		return err
	}
}
