package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/tehrelt/mu-lib/rmqmanager"
	"github.com/tehrelt/mu-lib/sl"
)

var (
	host string = "localhost"
	port int    = 5672
	user string = "guest"
	pass string = "guest"

	exchange string = "tests"
	queue    string = "test-queue"

	interval time.Duration = time.Second
)

func main() {
	suite := NewSuite()

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	go func() {
		produce(ctx, suite)
	}()

	go func() {
		consume(ctx, suite)
	}()

	<-ctx.Done()
}

func produce(ctx context.Context, suite *Suite) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	count := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			suite.manager.Publish(ctx, exchange, queue, fmt.Appendf(nil, "message#%d", count))
			count++
		}
	}
}

func consume(ctx context.Context, suite *Suite) {
	messages, err := suite.manager.Consume(ctx, queue)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		select {
		case <-ctx.Done():

			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			fmt.Printf("consuming event: %s\n", string(msg.Body))
			msg.Ack(false)
		}
	}
}

type Suite struct {
	manager *rmqmanager.RabbitMqManager
}

func _amqp() (*amqp091.Channel, func(), error) {
	cs := fmt.Sprintf("amqp://%s:%s@%s:%d/", user, pass, host, port)

	conn, err := amqp091.Dial(cs)
	if err != nil {
		return nil, nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, func() {
			conn.Close()
		}, err
	}

	closefn := func() {
		defer conn.Close()
		defer channel.Close()
	}

	if err := amqp_setup_exchange(channel, exchange, queue); err != nil {
		slog.Error("failed to setup exchange", sl.Err(err))
		return nil, closefn, err
	}

	return channel, closefn, nil
}

func amqp_setup_exchange(channel *amqp091.Channel, exchange string, queues ...string) error {

	log := slog.With(slog.String("exchange", exchange))
	log.Info("declaring exchange")
	if err := channel.ExchangeDeclare(exchange, "direct", false, true, false, false, nil); err != nil {
		slog.Error("failed to declare notifications queue", sl.Err(err))
		return err
	}

	for _, queueName := range queues {
		log.Info("declaring queue", slog.String("queue", queueName))
		queue, err := channel.QueueDeclare(queueName, false, true, false, false, nil)
		if err != nil {
			log.Error("failed to declare queue", sl.Err(err), slog.String("queue", queueName))
			return err
		}

		log.Info("binding queue", slog.String("queue", queueName))
		if err := channel.QueueBind(queue.Name, queueName, exchange, false, nil); err != nil {
			log.Error("failed to bind queue", sl.Err(err), slog.String("queue", queueName))
			return err
		}
	}

	return nil
}
func NewSuite() *Suite {
	ch, _, err := _amqp()
	if err != nil {
		panic(err)
	}
	return &Suite{
		manager: rmqmanager.New(ch),
	}
}
