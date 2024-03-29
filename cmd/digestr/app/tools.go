package app

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"mleku.dev/git/nostr/tag"
)

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
	if seedr.Present {
		return seedr.SeededGen.Intn(max)
	}
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		fmt.Printf("Error generating random integer: %v\n", err)
		return 0
	}
	return int(nBig.Int64())
}
