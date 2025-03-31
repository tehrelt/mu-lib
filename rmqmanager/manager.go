package rmqmanager

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rabbitmq/amqp091-go"
	"github.com/tehrelt/mu-lib/tracer"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type ConsumeFn func(ctx context.Context, msg amqp091.Delivery) error

type RabbitMqManager struct {
	ch *amqp091.Channel
}

func New(ch *amqp091.Channel) *RabbitMqManager {
	return &RabbitMqManager{
		ch: ch,
	}
}

func (r *RabbitMqManager) Consume(ctx context.Context, rk string, fn ConsumeFn) error {
	messages, err := r.ch.ConsumeWithContext(ctx, rk, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	for msg := range messages {
		headers := make(amqp091.Table)
		p := propagation.TraceContext{}
		ctx = p.Extract(ctx, &amqpTableCarrier{table: &headers})

		t := otel.Tracer(tracer.TracerKey)
		ctx, span := t.Start(ctx, fmt.Sprintf("Consume %s", rk))
		defer span.End()

		slog.Debug("consuming message", slog.Any("headers", msg.Headers))
		if err := fn(ctx, msg); err != nil {
			return err
		}
	}

	return nil
}

func (r *RabbitMqManager) Publish(ctx context.Context, exchange, rk string, msg []byte) error {

	headers := make(amqp091.Table)

	propagator := propagation.TraceContext{}
	propagator.Inject(ctx, &amqpTableCarrier{table: &headers})

	slog.Debug("publishing message", slog.Any("headers", headers))

	return r.ch.PublishWithContext(ctx, exchange, rk, false, false, amqp091.Publishing{
		Headers:     headers,
		ContentType: "application/json",
		Body:        msg,
	})
}
