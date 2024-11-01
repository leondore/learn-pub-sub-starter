package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueType int

const (
	Durable QueueType = iota
	Transient
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType QueueType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel: %v", err)
	}

	queue, err := ch.QueueDeclare(
		queueName,
		simpleQueueType == Durable,
		simpleQueueType == Transient,
		simpleQueueType == Transient,
		false,
		nil,
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue: %v", err)
	}

	if err = ch.QueueBind(queue.Name, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}

	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType QueueType,
	handler func(T),
) error {
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return fmt.Errorf("error creating new queue: %v", err)
	}

	delivery, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error initiating delivery: %v", err)
	}

	go func() {
		defer ch.Close()
		for msg := range delivery {
			var body T

			err := json.Unmarshal(msg.Body, &body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}

			handler(body)
			msg.Ack(false)
		}
	}()

	return nil
}
