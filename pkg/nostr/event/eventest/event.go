package eventest

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

var D = []*event.T{
	{
		ID:        "92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
		PubKey:    "e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
		Kind:      kind.EncryptedDirectMessage,
		CreatedAt: timestamp.T(1671028682),
		Tags: tags.T{tag.T{
			"p",
			"f8340b2bde651576b75af61aa26c80e13c65029f00f7f64004eece679bf7059f",
		}},
		// this invalidates the signature
		Content: "you say \"\"yes, I say {[no}]",
		Sig: "ed08d2dd5b0f7b6a3cdc74643d4adee3158ddede9cc848e8cd97630c097001ac" +
			"c2d052d2d3ec2b7ac4708b2314b797106d1b3c107322e61b5e5cc2116e099b79",
	},
	{
		ID:        "92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
		PubKey:    "e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
		Kind:      kind.EncryptedDirectMessage,
		CreatedAt: timestamp.T(1671028682),
		Tags: tags.T{tag.T{
			"p",
			"f8340b2bde651576b75af61aa26c80e13c65029f00f7f64004eece679bf7059f",
		}},
		Content: "you say yes, I say no",
		Sig: "ed08d2dd5b0f7b6a3cdc74643d4adee3158ddede9cc848e8cd97630c097001ac" +
			"c2d052d2d3ec2b7ac4708b2314b797106d1b3c107322e61b5e5cc2116e099b79",
	},
}
