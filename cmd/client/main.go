package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	// Connect to RabbitMQ
	connString := "amqp://guest:guest@localhost:5672/"
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

	// Open a channel
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

	// Declare the peril_direct exchange
	if err := declareDirectExchange(ch); err != nil {
		log.Fatalf("Failed to declare direct exchange: %v", err)
	}

	// Prompt for username
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Declare and bind a transient queue for pause messages
	pauseQueueName := fmt.Sprintf("pause.%s", username)
	_, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, pauseQueueName, routing.PauseKey, pubsub.TransientQueue)
	if err != nil {
		log.Fatalf("Failed to declare and bind pause queue: %v", err)
	}
	fmt.Printf("Pause queue %s declared and bound successfully!\n", pauseQueueName)

	// Create a new game state
	gameState := gamelogic.NewGameState(username)

	// Subscribe to pause messages
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, pauseQueueName, routing.PauseKey, pubsub.TransientQueue, handlerPause(gameState))
	if err != nil {
		log.Fatalf("Failed to subscribe to pause messages: %v", err)
	}

	// Declare and bind a transient queue for move messages
	moveQueueName := fmt.Sprintf("army_moves.%s", username)
	_, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, moveQueueName, "army_moves.*", pubsub.TransientQueue)
	if err != nil {
		log.Fatalf("Failed to declare and bind move queue: %v", err)
	}
	fmt.Printf("Move queue %s declared and bound successfully!\n", moveQueueName)

	// Subscribe to move messages
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, moveQueueName, "army_moves.*", pubsub.TransientQueue, handlerMove(gameState))
	if err != nil {
		log.Fatalf("Failed to subscribe to move messages: %v", err)
	}

	// Infinite REPL loop
	for {
		// Get user input
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue // If no input, skip to next iteration
		}

		// Handle commands
		switch words[0] {
		case "spawn":
			if len(words) < 3 {
				fmt.Println("Usage: spawn <location> <rank>")
				continue
			}

			unitID, err := gameState.CommandSpawn(words)
			if err != nil {
				fmt.Printf("Failed to spawn unit: %v\n", err)
				continue
			}

			fmt.Printf("Unit spawned successfully with ID: %d\n", unitID)

		case "move":
			if len(words) < 3 {
				fmt.Println("Usage: move <location> <unitID>")
				continue
			}
			move, err := gameState.CommandMove(words)
			if err != nil {
				fmt.Printf("Error moving unit: %v\n", err)
			} else {
				fmt.Printf("Move successful! Moved %d units to %s.\n", len(move.Units), move.ToLocation)

				// Open a new channel for publishing
				ch, err := conn.Channel()
				if err != nil {
					fmt.Printf("Failed to open channel for publishing move: %v\n", err)
					continue
				}
				defer ch.Close()

				// Publish the move
				err = pubsub.PublishJSON(ch, routing.ExchangePerilTopic, fmt.Sprintf("army_moves.%s", username), move)
				if err != nil {
					fmt.Printf("Failed to publish move: %v\n", err)
				} else {
					fmt.Println("Move published successfully.")
				}
			}

		case "status":
			gameState.CommandStatus()

		case "help":
			gamelogic.PrintClientHelp()

		case "spam":
			fmt.Println("Spamming not allowed yet!")

		case "quit":
			gamelogic.PrintQuit()
			return // Exit the loop and close the client

		default:
			fmt.Printf("Unknown command: %s\n", words[0])
		}
	}
}

func declareDirectExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		"peril_direct", // Exchange name
		"direct",       // Exchange type
		true,           // Durable
		false,          // Auto-deleted
		false,          // Internal
		false,          // No-wait
		nil,            // Arguments
	)
}

// handlerPause handles pause messages
func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ") // Ensure prompt is printed
		gs.HandlePause(ps)
		if ps.IsPaused {
			fmt.Println("Game is paused.")
		} else {
			fmt.Println("Game is resumed.")
		}
		return pubsub.Ack
	}
}

// handlerMove handles move messages
func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ") // Ensure prompt is printed
		outcome := gs.HandleMove(move)
		switch outcome {
		case gamelogic.MoveOutComeSafe, gamelogic.MoveOutcomeMakeWar:
			fmt.Printf("Move from %s detected: %d units to %s (Ack)\n", move.Player.Username, len(move.Units), move.ToLocation)
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			fmt.Printf("Move from %s ignored: Same player (NackDiscard)\n", move.Player.Username)
			return pubsub.NackDiscard
		default:
			fmt.Printf("Unexpected move outcome from %s: %v (NackDiscard)\n", move.Player.Username, outcome)
			return pubsub.NackDiscard
		}
	}
}
