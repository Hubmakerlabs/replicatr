package app

import (
	"encoding/json"
	"fmt"
	seededRand "math/rand"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/keys"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/timestamp"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Seedr struct {
	SeededGen *seededRand.Rand
	Present   bool
}

var seedr = Seedr{nil, false}

var kinds = []int{
	0, 1, 2, 3, 4, 5, 6, 7, 8, 15, 16, 40, 41, 42, 43, 44,
	1021, 1022, 1040, 1059, 1060, 1063, 1311, 1517, 1808,
	1971, 1984, 1985, 4550, 5000, 5999, 6000, 6999, 7000,
	9041, 9734, 9735, 9882, 10000, 10001, 10002, 10003,
	10004, 10005, 10006, 10007, 10015, 10030, 10096, 13194,
	20000, 21000, 22242, 23194, 23195, 24133, 27235, 30000,
	30001, 30002, 30003, 30004, 30008, 30009, 30015, 30017,
	30018, 30019, 30020, 30023, 30024, 30030, 30078, 30311,
	30315, 30402, 30403, 31922, 31923, 31924, 31925, 31989,
	31990, 32123, 34550, 39998, 40000,
}

func EventsTest(numEvents int, seed *int) error {
	if seed != nil {
		src := seededRand.NewSource(int64(*seed))
		seedr = Seedr{seededRand.New(src), true}
	}

	// Set up WebSocket connection
	wsURL := "ws://127.0.0.1:3334"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("Failed to connect to WebSocket: %s", err)
	}
	defer c.Close()

	for i := 0; i < numEvents; i++ {
		k := kinds[randomInt(len(kinds))]
		tags := generateTagsForKind(k)
		e := event.T{
			CreatedAt: timestamp.T(time.Now().Unix()),
			Kind:      kind.T(k),
			Tags:      tags,
			Content:   generateRandomContent(),
			Sig:       fmt.Sprintf("sig_placeholder_%d", i),
		}
		err := e.Sign(keys.GeneratePrivateKey())
		if err != nil {
			log.E.F("unable to create random event number %d out of %d: %v", i, numEvents, err)
			continue
		}

		wrappedEvent := []interface{}{"EVENT", e}

		jsonData, err := json.Marshal(wrappedEvent)
		if err != nil {
			fmt.Printf("Error marshaling event: %v\n", err)
			continue
		}

		err = c.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			log.E.F("Failed to send event %d out of %d through WebSocket: %v", i, numEvents, err)
		}

		c.SetReadDeadline(time.Now().Add(10 * time.Second)) // Set a 10-second read deadline
		_, message, err := c.ReadMessage()

		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				fmt.Printf("Connection closed normally.\n")
				break // Exit the loop if the connection is closed normally
			} else {
				return fmt.Errorf("Failed to read relay response from event %d out of %d from WebSocket or read timeout occurred: %v\n", i, numEvents, err)
			}
		}

		// The JSON array string you receive from the WebSocket
		jsonStr := string(message)

		// Define a variable to hold the parsed data
		var result []interface{}

		// Parse the JSON array
		err = json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			return fmt.Errorf("Error parsing JSON response from event %d out of %d:%v", i, numEvents, err)
		}

		// Check if the slice is not empty and then confirm its first element
		if len(result) > 0 {
			firstElement, ok := result[0].(string)
			if !ok {
				return fmt.Errorf("Type Assertion for first element of JSON response for event number %d out of %d failed", i, numEvents)
			} else {
				if firstElement == "OK" {
					fmt.Printf("Received OK %d out of %d\n", i, numEvents)
				} else {
					return fmt.Errorf("relay response message for event number %d out of %d was: %s", i, numEvents, firstElement)
				}
			}
		} else {
			fmt.Errorf("The JSON array response from event %d out of %d is empty.", i, numEvents)
		}
	}

	fmt.Printf("Event Test Successful! %d out of %d OK's received\n\n", numEvents, numEvents)
	return nil
}
