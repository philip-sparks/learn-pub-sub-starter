package main

import (
	"fmt"
	"log"

	// "os"
	// "os/signal"
	// "syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	// Print server help commands
	gamelogic.PrintServerHelp()

	// Declare connection string
	connString := "amqp://guest:guest@localhost:5672/"

	// Establish connection to RabbitMQ
	conn, err := amqp.Dial(connString)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer func() {
		fmt.Println("Closing connection to RabbitMQ...")
		if err := conn.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		}
	}()

	// Create a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer func() {
		fmt.Println("Closing RabbitMQ channel...")
		if err := ch.Close(); err != nil {
			log.Printf("Error closing RabbitMQ channel: %v", err)
		}
	}()

	// Declare the peril_topic exchange
	if err := declareTopicExchange(ch); err != nil {
		log.Fatalf("Failed to declare topic exchange: %v", err)
	}

	// Setup the dead letter exchange and queue
	if err := setupDeadLetterExchange(ch); err != nil {
		log.Fatalf("Failed to setup dead letter exchange and queue: %v", err)
	}

	// Declare and bind the durable game_logs queue to the peril_topic exchange
	queueName := "game_logs"
	routingKey := routing.GameLogSlug + ".*" // game_logs.*
	err = declareAndBindQueue(ch, queueName, routing.ExchangePerilTopic, routingKey)
	if err != nil {
		log.Fatalf("Failed to declare and bind queue: %v", err)
	}
	log.Printf("Queue %s declared and bound to exchange %s with routing key %s", queueName, routing.ExchangePerilTopic, routingKey)

	// Infinite REPL loop
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "pause":
			log.Println("Sending pause message...")
			err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
			if err != nil {
				log.Printf("Failed to publish pause message: %v", err)
			} else {
				log.Println("Pause message sent successfully.")
			}

		case "resume":
			log.Println("Sending resume message...")
			err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
			if err != nil {
				log.Printf("Failed to publish resume message: %v", err)
			} else {
				log.Println("Resume message sent successfully.")
			}

		case "quit":
			log.Println("Exiting...")
			return

		default:
			log.Printf("Unknown command: %s", words[0])
		}
	}
}

func declareTopicExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		"peril_topic", // Exchange name
		"topic",       // Exchange type
		true,          // Durable
		false,         // Auto-deleted
		false,         // Internal
		false,         // No-wait
		nil,           // Arguments
	)
}

// declareAndBindQueue declares a durable queue and binds it to an exchange with a given routing key
func declareAndBindQueue(ch *amqp.Channel, queueName, exchange, routingKey string) error {
	// Declare the durable queue
	_, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// Bind the queue to the exchange with the routing key
	err = ch.QueueBind(
		queueName,  // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s with routing key %s: %w", queueName, exchange, routingKey, err)
	}

	return nil
}

// setupDeadLetterExchange declares the dead letter exchange and its queue
func setupDeadLetterExchange(ch *amqp.Channel) error {
	// Declare the dead letter exchange
	err := ch.ExchangeDeclare(
		"dead_letter_exchange", // Name of the dead letter exchange
		"fanout",               // Type
		true,                   // Durable
		false,                  // Auto-deleted
		false,                  // Internal
		false,                  // No-wait
		nil,                    // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %w", err)
	}

	// Declare the dead-letter queue
	_, err = ch.QueueDeclare(
		"peril_dlq", // Queue name
		true,        // Durable
		false,       // Auto-delete
		false,       // Exclusive
		false,       // No-wait
		nil,         // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %w", err)
	}

	// Bind the dead-letter queue to the exchange
	err = ch.QueueBind(
		"peril_dlq",            // Queue name
		"",                     // Routing key (empty string matches all)
		"dead_letter_exchange", // Exchange name
		false,                  // No-wait
		nil,                    // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %w", err)
	}

	return nil
}
