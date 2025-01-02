package gamelogic

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand" // Importing the missing package
	"os"
	"strings"
)

// PrintServerHelp prints the list of available server commands
func PrintServerHelp() {
	fmt.Println("Possible commands:")
	fmt.Println("* pause   - Pauses the game")
	fmt.Println("* resume  - Resumes the game")
	fmt.Println("* quit    - Exits the server")
	fmt.Println("* help    - Prints this help message")
}

// PrintClientHelp prints the list of available client commands
func PrintClientHelp() {
	fmt.Println("Possible commands:")
	fmt.Println("* move <location> <unitID> <unitID> <unitID>...")
	fmt.Println("    example:")
	fmt.Println("    move asia 1")
	fmt.Println("* spawn <location> <rank>")
	fmt.Println("    example:")
	fmt.Println("    spawn europe infantry")
	fmt.Println("* status")
	fmt.Println("* spam <n>")
	fmt.Println("    example:")
	fmt.Println("    spam 5")
	fmt.Println("* quit")
	fmt.Println("* help")
}

// ClientWelcome prompts the user for their username
func ClientWelcome() (string, error) {
	fmt.Println("Welcome to the Peril client!")
	fmt.Println("Please enter your username:")
	words := GetInput()
	if len(words) == 0 {
		return "", errors.New("you must enter a username. goodbye")
	}
	username := words[0]
	fmt.Printf("Welcome, %s!\n", username)
	PrintClientHelp()
	return username, nil
}

// GetInput reads and returns a slice of input words from the user
func GetInput() []string {
	fmt.Print("> ")
	scanner := bufio.NewScanner(os.Stdin)
	scanned := scanner.Scan()
	if !scanned {
		return nil
	}
	line := scanner.Text()
	line = strings.TrimSpace(line)
	return strings.Fields(line)
}

// GetMaliciousLog generates a random log message (useful for testing)
func GetMaliciousLog() string {
	possibleLogs := []string{
		"Never interrupt your enemy when he is making a mistake.",
		"The hardest thing of all for a soldier is to retreat.",
		"A soldier will fight long and hard for a bit of colored ribbon.",
		"It is well that war is so terrible, otherwise we should grow too fond of it.",
		"The art of war is simple enough. Find out where your enemy is. Get at him as soon as you can. Strike him as hard as you can, and keep moving on.",
		"All warfare is based on deception.",
	}
	return possibleLogs[rand.Intn(len(possibleLogs))]
}

// PrintQuit prints a quit message for the client
func PrintQuit() {
	fmt.Println("I hate this game! (╯°□°)╯︵ ┻━┻")
}

// CommandStatus prints the game status
func (gs *GameState) CommandStatus() {
	if gs.isPaused() {
		fmt.Println("The game is paused.")
		return
	} else {
		fmt.Println("The game is not paused.")
	}

	p := gs.GetPlayerSnap()
	fmt.Printf("You are %s, and you have %d units.\n", p.Username, len(p.Units))
	for _, unit := range p.Units {
		fmt.Printf("* %v: %v, %v\n", unit.ID, unit.Location, unit.Rank)
	}
}
