package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
)

type Consumer struct {
	Conn *amqp.Connection
	Chan *amqp.Channel
	dsn  string
}

func CreateConsumer(dsn string) (*Consumer, error) {
	c := &Consumer{
		dsn: dsn,
	}

	var err error
	c.Conn, err = amqp.Dial(dsn)
	if err != nil {
		return nil, err
	}

	c.Chan, err = c.Conn.Channel()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Consumer) StartConsume(exchangeName, queueName, routingKey, consume string) (<-chan amqp.Delivery, error) {
	err := c.Chan.ExchangeDeclare(exchangeName, "direct", true,
		false, false,
		false, nil)
	if err != nil {
		return nil, err
	}

	_, err = c.Chan.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = c.Chan.QueueBind(
		queueName,    // queue name
		routingKey,   // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	msg, err := c.Chan.Consume(
		queueName, // queue
		consume,   // consume
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (c *Consumer) Shutdown() error {
	if c == nil || c.Conn == nil {
		return nil
	}

	err := c.Chan.Close()
	if err != nil {
		return err
	}
	err = c.Conn.Close()
	if err != nil {
		return err
	}

	c.Chan = nil
	c.Conn = nil

	return nil
}
