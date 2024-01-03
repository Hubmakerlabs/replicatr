package binary

import (
	"encoding/json"
	"testing"

	"github.com/mailru/easyjson"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr"
)

func BenchmarkBinaryEncoding(b *testing.B) {
	events := make([]*nostr.Event, len(normalEvents))
	binaryEvents := make([]*Event, len(normalEvents))
	for i, jevt := range normalEvents {
		evt := &nostr.Event{}
		json.Unmarshal([]byte(jevt), evt)
		events[i] = evt
		binaryEvents[i] = BinaryEvent(evt)
	}

	b.Run("easyjson.Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events {
				easyjson.Marshal(evt)
			}
		}
	})

	b.Run("binary.Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events {
				Marshal(evt)
			}
		}
	})

	b.Run("binary.MarshalBinary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range binaryEvents {
				MarshalBinary(bevt)
			}
		}
	})
}

func BenchmarkBinaryDecoding(b *testing.B) {
	events := make([][]byte, len(normalEvents))
	for i, jevt := range normalEvents {
		evt := &nostr.Event{}
		json.Unmarshal([]byte(jevt), evt)
		bevt, _ := Marshal(evt)
		events[i] = bevt
	}

	b.Run("easyjson.Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, jevt := range normalEvents {
				evt := &nostr.Event{}
				err := easyjson.Unmarshal([]byte(jevt), evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
			}
		}
	})

	b.Run("binary.Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range events {
				evt := &nostr.Event{}
				err := Unmarshal(bevt, evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
			}
		}
	})

	b.Run("binary.UnmarshalBinary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range events {
				evt := &Event{}
				err := UnmarshalBinary(bevt, evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
			}
		}
	})

	b.Run("easyjson.Unmarshal+sig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, nevt := range normalEvents {
				evt := &nostr.Event{}
				err := easyjson.Unmarshal([]byte(nevt), evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
				evt.CheckSignature()
			}
		}
	})

	b.Run("binary.Unmarshal+sig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range events {
				evt := &nostr.Event{}
				err := Unmarshal(bevt, evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
				evt.CheckSignature()
			}
		}
	})
}
