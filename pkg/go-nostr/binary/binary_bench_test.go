package binary

import (
	"encoding/json"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/mailru/easyjson"
)

func BenchmarkBinaryEncoding(b *testing.B) {
	events := make([]*event.T, len(normalEvents))
	binaryEvents := make([]*Event, len(normalEvents))
	for i, jevt := range normalEvents {
		evt := &event.T{}
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
		evt := &event.T{}
		json.Unmarshal([]byte(jevt), evt)
		bevt, _ := Marshal(evt)
		events[i] = bevt
	}

	b.Run("easyjson.Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, jevt := range normalEvents {
				evt := &event.T{}
				e := easyjson.Unmarshal([]byte(jevt), evt)
				if e != nil {
					b.Fatalf("failed to unmarshal: %s", e)
				}
			}
		}
	})

	b.Run("binary.Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range events {
				evt := &event.T{}
				e := Unmarshal(bevt, evt)
				if e != nil {
					b.Fatalf("failed to unmarshal: %s", e)
				}
			}
		}
	})

	b.Run("binary.UnmarshalBinary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range events {
				evt := &Event{}
				e := UnmarshalBinary(bevt, evt)
				if e != nil {
					b.Fatalf("failed to unmarshal: %s", e)
				}
			}
		}
	})

	b.Run("easyjson.Unmarshal+sig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, nevt := range normalEvents {
				evt := &event.T{}
				e := easyjson.Unmarshal([]byte(nevt), evt)
				if e != nil {
					b.Fatalf("failed to unmarshal: %s", e)
				}
				evt.CheckSignature()
			}
		}
	})

	b.Run("binary.Unmarshal+sig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range events {
				evt := &event.T{}
				e := Unmarshal(bevt, evt)
				if e != nil {
					b.Fatalf("failed to unmarshal: %s", e)
				}
				evt.CheckSignature()
			}
		}
	})
}
