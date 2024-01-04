package envelope

import (
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/closed"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/ptr"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/req"
)

func TestParseMessage(t *testing.T) {
	testCases := []struct {
		Name             string
		Message          []byte
		ExpectedEnvelope envelopes.Envelope
	}{
		{
			Name:             "nil",
			Message:          nil,
			ExpectedEnvelope: nil,
		},
		{
			Name:             "invalid string",
			Message:          []byte("invalid input"),
			ExpectedEnvelope: nil,
		},
		{
			Name:             "invalid string with a comma",
			Message:          []byte("invalid, input"),
			ExpectedEnvelope: nil,
		},
		{
			Name:             "CLOSED envelope",
			Message:          []byte(`["CLOSED",":1","error: we are broken"]`),
			ExpectedEnvelope: &closed.Envelope{SubscriptionID: ":1", Reason: "error: we are broken"},
		},
		{
			Name:             "AUTH envelope",
			Message:          []byte(`["AUTH","bisteka"]`),
			ExpectedEnvelope: &auth.Envelope{Challenge: ptr.Ptr("bisteka")},
		},
		{
			Name:             "REQ envelope",
			Message:          []byte(`["REQ","million", {"kinds": [1]}, {"kinds": [30023 ], "#d": ["buteko",    "batuke"]}]`),
			ExpectedEnvelope: &req.ReqEnvelope{SubscriptionID: "million", Filters: filter.Filters{{Kinds: []int{1}}, {Kinds: []int{30023}, Tags: filter.TagMap{"d": []string{"buteko", "batuke"}}}}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			env := ParseMessage(testCase.Message)
			if testCase.ExpectedEnvelope == nil && env == nil {
				return
			}
			if testCase.ExpectedEnvelope == nil && env != nil {
				t.Fatalf("expected nil but got %v\n", env)
			}
			if testCase.ExpectedEnvelope.String() != env.String() {
				t.Fatalf("unexpected output:\n     %s\n  != %s", testCase.ExpectedEnvelope, env)
			}
		})
	}
}

