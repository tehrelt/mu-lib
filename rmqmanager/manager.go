package rmqmanager

import (
	"context"
	"log/slog"

	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/propagation"
)

type RabbitMqManager struct {
	ch *amqp091.Channel
}

func New(ch *amqp091.Channel) *RabbitMqManager {
	return &RabbitMqManager{
		ch: ch,
	}
}

func (r *RabbitMqManager) Publish(ctx context.Context, exchange, rk string, msg []byte) error {

	headers := make(amqp091.Table)

	carrier := propagation.MapCarrier{}
	propagator := propagation.TraceContext{}
	propagator.Inject(ctx, carrier)

	for k, v := range carrier {
		headers[k] = v
	}

	slog.Debug("publishing message", slog.Any("headers", headers))

	return r.ch.PublishWithContext(ctx, exchange, rk, false, false, amqp091.Publishing{
		Headers:     headers,
		ContentType: "application/json",
		Body:        msg,
	})
}
