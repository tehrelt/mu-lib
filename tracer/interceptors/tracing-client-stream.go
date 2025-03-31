package interceptors

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// tracingClientStream wraps grpc.ClientStream to properly close the span
type tracingClientStream struct {
	grpc.ClientStream
	span trace.Span
}

func (s *tracingClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (s *tracingClientStream) SendMsg(m interface{}) error {
	err := s.ClientStream.SendMsg(m)
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (s *tracingClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (s *tracingClientStream) Header() (metadata.MD, error) {
	md, err := s.ClientStream.Header()
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
	return md, err
}

func (s *tracingClientStream) Trailer() metadata.MD {
	return s.ClientStream.Trailer()
}

func (s *tracingClientStream) Context() context.Context {
	return s.ClientStream.Context()
}
