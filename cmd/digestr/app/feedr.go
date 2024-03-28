package app

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/gorilla/websocket"
)

func FeedEvents() {
	eventsFilePath := "generated_events.jsonl"
	file, err := os.Open(eventsFilePath)
	if err != nil {
		log.E.F("Failed to open file: %s", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Set up a WebSocket connection to the local replicatr relay
	wsURL := "ws://127.0.0.1:3334"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.E.F("Failed to connect to WebSocket: %s", err)
	}
	defer c.Close()

	for scanner.Scan() {
		line := scanner.Text()

		var rawMessage json.RawMessage
		if err := json.Unmarshal([]byte(line), &rawMessage); err != nil {
			log.E.F("Error unmarshaling wrapped event as raw JSON: %s", err)
		}
		err = c.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			log.E.F("Failed to send raw event through WebSocket: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.E.F("Error reading from file: %s", err)
	}
}
