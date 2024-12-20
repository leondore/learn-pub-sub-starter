package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueType int

type AckType int

const (
	Durable QueueType = iota
	Transient
)

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
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
		amqp.Table{"x-dead-letter-exchange": "peril_dlx"},
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
	handler func(T) AckType,
) error {
	return subscribe[T](
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
		handler,
		func(data []byte) (T, error) {
			var val T
			err := json.Unmarshal(data, &val)
			return val, err
		},
	)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType QueueType,
	handler func(T) AckType,
) error {
	return subscribe[T](
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
		handler,
		func(data []byte) (T, error) {
			var val T
			buffer := bytes.NewBuffer(data)
			decoder := gob.NewDecoder(buffer)
			err := decoder.Decode(&val)
			return val, err
		},
	)
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType QueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
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
			body, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}

			ackType := handler(body)
			switch ackType {
			case Ack:
				msg.Ack(false)

			case NackRequeue:
				msg.Nack(false, true)

			case NackDiscard:
				msg.Nack(false, false)

			}
		}
	}()

	return nil
}
