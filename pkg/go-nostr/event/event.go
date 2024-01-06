package event

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/escape"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/ec"
	"github.com/Hubmakerlabs/replicatr/pkg/ec/schnorr"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

type T struct {
	ID        string              `json:"id"`
	PubKey    string      `json:"pubkey"`
	CreatedAt timestamp.T `json:"created_at"`
	Kind      int         `json:"kind"`
	Tags      tags.Tags           `json:"tags"`
	Content   string              `json:"content"`
	Sig       string              `json:"sig"`

	// anything here will be mashed together with the main event object when serializing
	extra map[string]any
}

const (
	KindProfileMetadata             int = 0
	KindTextNote                    int = 1
	KindRecommendServer             int = 2
	KindContactList                 int = 3
	KindEncryptedDirectMessage      int = 4
	KindDeletion                    int = 5
	KindRepost                      int = 6
	KindReaction                    int = 7
	KindSimpleGroupChatMessage      int = 9
	KindSimpleGroupThread           int = 11
	KindSimpleGroupReply            int = 12
	KindChannelCreation             int = 40
	KindChannelMetadata             int = 41
	KindChannelMessage              int = 42
	KindChannelHideMessage          int = 43
	KindChannelMuteUser             int = 44
	KindFileMetadata                int = 1063
	KindSimpleGroupAddUser          int = 9000
	KindSimpleGroupRemoveUser       int = 9001
	KindSimpleGroupEditMetadata     int = 9002
	KindSimpleGroupAddPermission    int = 9003
	KindSimpleGroupRemovePermission int = 9004
	KindSimpleGroupDeleteEvent      int = 9005
	KindSimpleGroupEditGroupStatus  int = 9006
	KindZapRequest                  int = 9734
	KindZap                         int = 9735
	KindMuteList                    int = 10000
	KindPinList                     int = 10001
	KindRelayListMetadata           int = 10002
	KindNWCWalletInfo               int = 13194
	KindClientAuthentication        int = 22242
	KindNWCWalletRequest            int = 23194
	KindNWCWalletResponse           int = 23195
	KindNostrConnect                int = 24133
	KindCategorizedPeopleList       int = 30000
	KindCategorizedBookmarksList    int = 30001
	KindProfileBadges               int = 30008
	KindBadgeDefinition             int = 30009
	KindStallDefinition             int = 30017
	KindProductDefinition           int = 30018
	KindArticle                     int = 30023
	KindApplicationSpecificData     int = 30078
	KindSimpleGroupMetadata         int = 39000
	KindSimpleGroupAdmins           int = 39001
	KindSimpleGroupMembers          int = 39002
)

// T Stringer interface, just returns the raw JSON as a string.
func (evt T) String() string {
	j, _ := easyjson.Marshal(evt)
	return string(j)
}

// GetID serializes and returns the event ID as a string.
func (evt *T) GetID() string {
	h := sha256.Sum256(evt.Serialize())
	return hex.EncodeToString(h[:])
}

// Serialize outputs a byte array that can be hashed/signed to identify/authenticate.
// JSON encoding as defined in RFC4627.
func (evt *T) Serialize() []byte {
	// the serialization process is just putting everything into a JSON array
	// so the order is kept. See NIP-01
	dst := make([]byte, 0)

	// the header portion is easy to serialize
	// [0,"pubkey",created_at,kind,[
	dst = append(dst, []byte(
		fmt.Sprintf(
			"[0,\"%s\",%d,%d,",
			evt.PubKey,
			evt.CreatedAt,
			evt.Kind,
		))...)

	// tags
	dst = evt.Tags.MarshalTo(dst)
	dst = append(dst, ',')

	// content needs to be escaped in general as it is user generated.
	dst = escape.String(dst, evt.Content)
	dst = append(dst, ']')

	return dst
}

// CheckSignature checks if the signature is valid for the id
// (which is a hash of the serialized event content).
// returns an error if the signature itself is invalid.
func (evt T) CheckSignature() (bool, error) {
	// read and check pubkey
	pk, e := hex.DecodeString(evt.PubKey)
	if e != nil {
		return false, fmt.Errorf("event pubkey '%s' is invalid hex: %w", evt.PubKey, e)
	}

	pubkey, e := schnorr.ParsePubKey(pk)
	if e != nil {
		return false, fmt.Errorf("event has invalid pubkey '%s': %w", evt.PubKey, e)
	}

	// read signature
	s, e := hex.DecodeString(evt.Sig)
	if e != nil {
		return false, fmt.Errorf("signature '%s' is invalid hex: %w", evt.Sig, e)
	}
	sig, e := schnorr.ParseSignature(s)
	if e != nil {
		return false, fmt.Errorf("failed to parse signature: %w", e)
	}

	// check signature
	hash := sha256.Sum256(evt.Serialize())
	return sig.Verify(hash[:], pubkey), nil
}

// Sign signs an event with a given privateKey.
func (evt *T) Sign(privateKey string, signOpts ...schnorr.SignOption) error {
	s, e := hex.DecodeString(privateKey)
	if e != nil {
		return fmt.Errorf("Sign called with invalid private key '%s': %w", privateKey, e)
	}

	if evt.Tags == nil {
		evt.Tags = make(tags.Tags, 0)
	}

	sk, pk := btcec.PrivKeyFromBytes(s)
	pkBytes := pk.SerializeCompressed()
	evt.PubKey = hex.EncodeToString(pkBytes[1:])

	h := sha256.Sum256(evt.Serialize())
	sig, e := schnorr.Sign(sk, h[:], signOpts...)
	if e != nil {
		return e
	}

	evt.ID = hex.EncodeToString(h[:])
	evt.Sig = hex.EncodeToString(sig.Serialize())

	return nil
}

type Envelope struct {
	SubscriptionID *string
	T
}

var _ envelopes.E = (*Envelope)(nil)

func (_ Envelope) Label() string { return "EVENT" }

func (v *Envelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	switch len(arr) {
	case 2:
		return easyjson.Unmarshal([]byte(arr[1].Raw), &v.T)
	case 3:
		v.SubscriptionID = &arr[1].Str
		return easyjson.Unmarshal([]byte(arr[2].Raw), &v.T)
	default:
		return fmt.Errorf("failed to decode EVENT envelope")
	}
}

func (v Envelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["EVENT",`)
	if v.SubscriptionID != nil {
		w.RawString(`"` + *v.SubscriptionID + `",`)
	}
	v.MarshalEasyJSON(&w)
	w.RawString(`]`)
	return w.BuildBytes()
}
