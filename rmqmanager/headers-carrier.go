package rmqmanager

import "github.com/rabbitmq/amqp091-go"

type amqpTableCarrier struct {
	table *amqp091.Table
}

func (c *amqpTableCarrier) Get(key string) string {
	val, ok := (*c.table)[key]
	if !ok {
		return ""
	}

	return val.(string)
}

func (c *amqpTableCarrier) Set(key, value string) {
	(*c.table)[key] = value
}

func (c *amqpTableCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.table))
	for k := range *c.table {
		keys = append(keys, k)
	}
	return keys
}
