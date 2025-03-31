package rmqmanager_test

import (
	"github.com/rabbitmq/amqp091-go"
	"github.com/tehrelt/mu-lib/rmqmanager"
)

type RMQManagerTestSuite struct {
	manager *rmqmanager.RabbitMqManager
}

func NewSuite(url string) (*RMQManagerTestSuite, func(), error) {

	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, func() { conn.Close() }, err
	}

	manager := rmqmanager.New(ch)

	return &RMQManagerTestSuite{
			manager: manager,
		}, func() {
			defer conn.Close()
			defer ch.Close()
		}, nil

}
