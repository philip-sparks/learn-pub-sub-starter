package routing

// Constants for routing keys
const (
	ArmyMovesPrefix        = "army_moves"
	WarRecognitionsPrefix  = "war"
	PauseKey               = "pause"
	GameLogSlug            = "game_logs"
)

// Constants for RabbitMQ exchanges
const (
	ExchangePerilDirect = "peril_direct"
	ExchangePerilTopic  = "peril_topic"
)

// PlayingState represents the game state for pause/resume messages
//type PlayingState struct {
//	IsPaused bool `json:"isPaused"`
//}
