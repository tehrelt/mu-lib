package rmqmanager

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rabbitmq/amqp091-go"
	"github.com/tehrelt/mu-lib/tracer"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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

type TracedDelivery struct {
	ctx  context.Context
	span trace.Span
	amqp091.Delivery
}

func (d *TracedDelivery) Ack(multiple bool) error {
	defer d.end()
	return d.Delivery.Ack(multiple)
}

func (d *TracedDelivery) Nack(multiple, requeue bool) error {
	defer d.end()
	return d.Delivery.Nack(multiple, requeue)
}

func (d *TracedDelivery) end() {
	fmt.Println("ending span")
	d.span.End()
}

func (d *TracedDelivery) Context() context.Context {
	return d.ctx
}

func (d *TracedDelivery) Reject(requeue bool) error {
	defer d.end()
	return d.Delivery.Reject(requeue)
}

func (r *RabbitMqManager) Consume(ctx context.Context, rk string) (<-chan *TracedDelivery, error) {
	messages, err := r.ch.ConsumeWithContext(ctx, rk, "", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	ch := make(chan *TracedDelivery)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case msg := <-messages:
				p := propagation.TraceContext{}
				ctx = p.Extract(ctx, &amqpTableCarrier{table: &msg.Headers})

				t := otel.Tracer(tracer.TracerKey)
				ctx, span := t.Start(ctx, fmt.Sprintf("Consume %s", rk))

				ch <- &TracedDelivery{
					ctx:      ctx,
					span:     span,
					Delivery: msg,
				}
			}
		}
	}()

	return ch, nil
}

func (r *RabbitMqManager) Publish(ctx context.Context, exchange, rk string, msg []byte) error {

	t := otel.Tracer(tracer.TracerKey)
	ctx, span := t.Start(ctx, fmt.Sprintf("Publish %s", rk))
	defer span.End()

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
