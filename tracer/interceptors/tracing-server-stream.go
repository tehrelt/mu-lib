package interceptors

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// tracedServerStream wraps grpc.ServerStream for tracing
type tracedServerStream struct {
	grpc.ServerStream
	ctx  context.Context
	span trace.Span
}

func (s *tracedServerStream) Context() context.Context {
	return s.ctx
}

func (s *tracedServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err != nil {
		s.span.RecordError(err)
	}
	return err
}

func (s *tracedServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err != nil {
		s.span.RecordError(err)
	}
	return err
}
