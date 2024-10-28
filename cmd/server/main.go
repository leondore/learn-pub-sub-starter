package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	CONN_URL = "amqp://guest:guest@localhost:5672/"
)

func main() {
	conn, err := amqp.Dial(CONN_URL)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	fmt.Println("Peril game server successfully connected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not open a channel: %v", err)
	}

	err = pubsub.PublishJSON(
		ch,
		routing.ExchangePerilDirect,
		routing.PauseKey,
		routing.PlayingState{
			IsPaused: true,
		},
	)
	if err != nil {
		log.Fatalf("error sending message: %v", err)
	}

	fmt.Println("Pause message sent")
}
