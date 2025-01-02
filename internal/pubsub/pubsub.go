package pubsub

import (
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

// PublishJSON marshals a value to JSON and publishes it to the given exchange with a routing key.
func PublishJSON[T any](ch *amqp091.Channel, exchange, key string, val T) error {
	// Marshal value to JSON
	body, err := json.Marshal(val)
	if err != nil {
		return err
	}

	// Publish message using the older Publish method
	return ch.Publish(
		exchange, // Exchange
		key,      // Routing key
		false,    // Mandatory
		false,    // Immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// Enum for queue type
const (
	DurableQueue   = 0
	TransientQueue = 1
)

// DeclareAndBind declares a queue and binds it to an exchange.
func DeclareAndBind(
	conn *amqp091.Connection,
	exchange, queueName, key string,
	simpleQueueType int,
) (*amqp091.Channel, amqp091.Queue, error) {
	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp091.Queue{}, fmt.Errorf("failed to create channel: %w", err)
	}

	// Determine queue properties based on type
	durable := simpleQueueType == DurableQueue
	autoDelete := simpleQueueType == TransientQueue
	// exclusive := simpleQueueType == TransientQueue
	exclusive := false

	// Add dead letter exchange configuration
	args := amqp091.Table{
		"x-dead-letter-exchange": "dead_letter_exchange", // Dead letter exchange name
	}

	// Declare the queue
	queue, err := ch.QueueDeclare(
		queueName,  // name
		durable,    // durable
		autoDelete, // autoDelete
		exclusive,  // exclusive
		false,      // noWait
		args,       // arguments with dead letter exchange
	)
	if err != nil {
		return nil, amqp091.Queue{}, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind the queue to the exchange
	err = ch.QueueBind(
		queue.Name, // queue name
		key,        // routing key
		exchange,   // exchange
		false,      // noWait
		nil,        // args
	)
	if err != nil {
		return nil, amqp091.Queue{}, fmt.Errorf("failed to bind queue: %w", err)
	}

	return ch, queue, nil
}

// AckType defines the type of acknowledgment for a message
type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

// SubscribeJSON subscribes to a queue and processes JSON messages with the given handler
func SubscribeJSON[T any](
	conn *amqp091.Connection,
	exchange, queueName, key string,
	simpleQueueType int,
	handler func(T) AckType,
) error {
	// Use DeclareAndBind to ensure the queue exists and is bound
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return fmt.Errorf("failed to declare and bind queue: %w", err)
	}

	// Consume messages from the queue
	msgs, err := ch.Consume(
		queue.Name, // Queue name
		"",         // Consumer (auto-generated)
		false,      // Auto-ack
		false,      // Exclusive
		false,      // No-local
		false,      // No-wait
		nil,        // Args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	// Start a goroutine to process messages
	go func() {
		for msg := range msgs {
			var payload T
			// Unmarshal the message body into the generic type T
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				fmt.Printf("Error unmarshaling message: %v\n", err)
				continue
			}

			// Call the handler with the unmarshaled payload
			ackType := handler(payload)

			// Handle acknowledgment based on the returned ackType
			switch ackType {
			case Ack:
				fmt.Println("Acknowledging message...")
				if err := msg.Ack(false); err != nil {
					fmt.Printf("Failed to acknowledge message: %v\n", err)
				}
			case NackRequeue:
				fmt.Println("Nack with requeue...")
				if err := msg.Nack(false, true); err != nil {
					fmt.Printf("Failed to nack with requeue: %v\n", err)
				}
			case NackDiscard:
				fmt.Println("Nack with discard...")
				if err := msg.Nack(false, false); err != nil {
					fmt.Printf("Failed to nack with discard: %v\n", err)
				} else {
					fmt.Println("Message discarded successfully.")
				}
			}
		}
	}()

	return nil
}
