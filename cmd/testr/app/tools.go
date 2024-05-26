package app

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mrand "math/rand"

	"github.com/nbd-wtf/go-nostr"
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

func generateTagsForKind(kind int) []nostr.Tag {
	switch kind {
	case 1:
		return []nostr.Tag{{"e", "reply-to-event-id"}, {"#hashtag", "exampleHashtag"}}
	case 4:
		return []nostr.Tag{{"p", "recipient-public-key"}, {"dm", "1"}}
	// will add more cases later
	default:
		return []nostr.Tag{{"e", "5c83da77af1dec6d7289834998ad7aafbd9e2191396d75ec3cc27f5a77226f36", "wss://nostr.example.com"}}
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

// randomSubset selects a random subset of the given slice
func randomSubset(slice []string) []string {
	shuffled := make([]string, len(slice))
	copy(shuffled, slice)
	mrand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	count := mrand.Intn(len(shuffled)) + 1 // Ensure at least one element is chosen
	return shuffled[:count]
}

// generateRandomFilter creates a random filter given arrays of authors and IDs
func generateRandomFilter(authors, ids []string) nostr.Filter {
	return nostr.Filter{
		IDs:     randomSubset(ids),
		Authors: randomSubset(authors),
	}
}
