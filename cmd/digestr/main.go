package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	"github.com/gorilla/websocket"
)

func main() {
	eventsFilePath := "cmd/eventGenerator/generated_events.jsonl"
	file, err := os.Open(eventsFilePath)
	if err != nil {
		log.Fatalf("Failed to open file: %s", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Set up a WebSocket connection to the local replicatr relay
	wsURL := "ws://0.0.0.0:3334"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket: %s", err)
	}
	defer c.Close()

	for scanner.Scan() {
		line := scanner.Text()

		var rawMessage json.RawMessage
		if err := json.Unmarshal([]byte(line), &rawMessage); err != nil {
			log.Fatalf("Error unmarshaling wrapped event as raw JSON: %s", err)
		}
		err = c.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			log.Printf("Failed to send raw event through WebSocket: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading from file: %s", err)
	}
}
