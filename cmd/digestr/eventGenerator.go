package digestr

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"mleku.dev/git/slog"

	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/keys"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/tag"
	"mleku.dev/git/nostr/timestamp"
)

var log, _ = slog.New(os.Stderr)

// Define kinds based on the provided Perl code.
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

func generateEvents() {
	numEvents := 1000 // Number of events to generate
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.E.F("Failed to get current file path")
		return
	}
	currentDir := filepath.Dir(currentFile)
	outputFile := filepath.Join(currentDir, "generated_events.jsonl")

	file, err := os.Create(outputFile)
	if err != nil {
		log.E.F("Could not create file '%s': %v", outputFile, err)
		return
	}
	defer file.Close()

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
			log.E.F("unable to create random event number %d: %v", i, err)
		}

		wrappedEvent := []interface{}{"EVENT", e}

		jsonData, err := json.Marshal(wrappedEvent)
		if err != nil {
			fmt.Printf("Error marshaling event: %v\n", err)
			continue
		}

		file.WriteString(string(jsonData) + "\n")
	}

	fmt.Printf("Generated %d events to %s\n", numEvents, outputFile)
}

func generateRandomContent() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 ,.!?"
	minLength, maxLength := 1, 500
	length := randomInt(maxLength-minLength) + minLength
	content := make([]byte, length)
	for i := range content {
		content[i] = chars[randomInt(len(chars))]
	}
	return string(content)
}

func generateTagsForKind(kind int) []tag.T {
	switch kind {
	case 1:
		return []tag.T{{"e", "reply-to-event-id"}, {"#hashtag", "exampleHashtag"}}
	case 4:
		return []tag.T{{"p", "recipient-public-key"}, {"dm", "1"}}
	// will add more cases later
	default:
		return []tag.T{{"e", "5c83da77af1dec6d7289834998ad7aafbd9e2191396d75ec3cc27f5a77226f36", "wss://nostr.example.com"}}
	}
}

func randomInt(max int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		fmt.Printf("Error generating random integer: %v\n", err)
		return 0
	}
	return int(nBig.Int64())
}
