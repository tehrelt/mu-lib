package interceptors

import (
	"context"
	"errors"
	"io"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// tracingClientStream wraps grpc.ClientStream to properly close the span
type tracingClientStream struct {
	grpc.ClientStream
	span  trace.Span
	start time.Time
}

func (s *tracingClientStream) end() {
	s.span.End()
	s.span.SetAttributes(
		attribute.Int64("rpc.duration_ms", time.Since(s.start).Milliseconds()),
	)
}

func (s *tracingClientStream) catchErr(err error) {
	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())

}

func (s *tracingClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		if errors.Is(err, io.EOF) {
			s.end()
			return err
		}

		s.catchErr(err)
	}
	return err
}

func (s *tracingClientStream) SendMsg(m interface{}) error {
	err := s.ClientStream.SendMsg(m)
	if err != nil {
		s.catchErr(err)
	}
	return err
}

func (s *tracingClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	if err != nil {
		s.catchErr(err)
	}
	s.end()
	return err
}

func (s *tracingClientStream) Header() (metadata.MD, error) {
	md, err := s.ClientStream.Header()
	if err != nil {
		s.catchErr(err)
	}
	return md, err
}

func (s *tracingClientStream) Trailer() metadata.MD {
	return s.ClientStream.Trailer()
}

func (s *tracingClientStream) Context() context.Context {
	return s.ClientStream.Context()
}
