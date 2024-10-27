package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	fmt.Println("connection to RabbitMQ server was successful")

	// Wait for user termination signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	fmt.Println("closing connection to RabbitMQ server")
}
