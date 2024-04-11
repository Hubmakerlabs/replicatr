# object

This library is a replacement for the use of `encoding/json` for tagged structs
that is both performant and maintains the order of fields.

### the map field ordering problem

The reason why there is a problem with the `encoding/json` library is that
unless you unmarshal json data into a json tagged struct, it will result in
a `map[string]interface{}` which is a set-style unique key hash table - because
keys may not be repeated, they are not kept in order, and various conditions
related to the key hash generation change the sort order of the table.

In recent versions of Go standard library, it looks like the fields may be
sorted lexicographically, at least in some cases. This is not an acceptable
output either.

Unfortunately, many implementations of JSON encoding depend on ordering, and in
the case of `nostr` the signatures on events must be performed on objects that
have an ordering that will be reproduced by any other client from the decoded
data structure, and if this ordering ever changes, it cannot produce the same
hash.

For the most part this only affects the `Event` type but due to the way that the
Go json encoder disagrees on ordering due to the rational argument that the
object is a set of keys with attached values, versus a struct, which is a
different beast, which has a fixed ordering and ordering is critical because
access to the fields is based on offsets of the symbols referring to the fields.
If the wizards in the nostr specification collective make more things that must
be canonically ordered and explicitly include JSON objects again Go will be
disadvantaged as an implementation language, despite its far superior capability
to actually handle the data at extreme volume and volatility of rates with low
latency.

### the performance hit of reflect

A second issue we aim to deal with in this library is minimising the use of
reflect. For this reason, the creation of the object form of a json tagged
struct has to be done manually for each struct that will become a JSON object (
wrapped in braces `{}`).

This is the `Event`type and its associated object generator:

```go
// Event is the primary datatype of nostr. This is the form of the structure
// that defines its JSON string based format.
type Event struct {
// ID is the SHA256 hash of the canonical encoding of the eventenvelope
ID string `json:"id"`
// PubKey is the public key of the eventenvelope creator in *hexadecimal* format
PubKey string `json:"pubkey"`
// CreatedAt is the UNIX timestamp of the eventenvelope according to the eventenvelope
// creator (never trust a timestamp!)
CreatedAt timestamp.T `json:"created_at"`
// Kind is the nostr protocol code for the type of eventenvelope. See kind.T
Kind kind.T `json:"kind"`
// Tags are a list of tags, which are a list of strings usually structured
// as a 3 layer scheme indicating specific features of an eventenvelope.
Tags tags.T `json:"tags"`
// Content is an arbitrary string that can contain anything, but usually
// conforming to a specification relating to the Kind and the Tags.
Content string `json:"content"`
// Sig is the signature on the ID hash that validates as coming from the
// Pubkey.
Sig string `json:"sig"`
}

func (ev *Event) ToObject() (o object.T) {
return object.T{
{"id", ev.ID},
{"pubkey", ev.PubKey},
{"created_at", ev.CreatedAt},
{"kind", ev.Kind},
{"tags", ev.Tags},
{"content", ev.Content},
{"sig", ev.Sig},
}
}
```

As you can see, creating the object generator is not a big job. The second
important task, which involves the `array` library that comes along with this
one, which properly handles `[]interface{}` slices, is used for producing the "
canonical" form that is hashed to derive the `Event.ID` that is then signed to
produce the `Event.Sig` - a BIP-340 compliant Schnorr signature.

```go
// ToCanonical returns a structure that provides a stringer that
// generates the canonical form used to generate the ID hash that can be signed.
func (ev *Event) ToCanonical() (o array.T) {
return array.T{0, ev.PubKey, ev.CreatedAt, ev.Kind, ev.Tags, ev.Content}
}
```

As you can see, this canonical form is a lot smaller and easy to manually
create, and note it uses an *ordered* slice type.

Being one of the most frequently encoded data objects in the `nostr` protocol,
as you can see, by using this library we completely avoid the use of reflection
and also avoid the problem of field ordering, and the workload and potential for
protocol compliance errors is basically zero.

Since the effort was made to build this order-respecting encoder for wire
encoded JSON, it also handles standard JSON tagged field hints, which again
relate to the loose typing scheme of Javascript, which is the `omitempty`
qualifier in the filed tag.

Unfortunately, due to the `array.T` being a `[]interface{}` and because future
proofing this JSON output to be correct for embedding both objects,
other `array` types and anything anyone can dream up and create a
runtime-generated complex data structure, to handle `omitempty` this requires
reflection in order to compare the value to the zero value or to check if a
nillable type is nil, or if the interface itself is nil, and then not add the
field to the `object.T` key/value slice.

`encoding/json` uses `reflect` by default and thus on the encoding side, several
libraries such as `easyjson` have been created, which automate producing
generators. It is precisely this use case that `object` aims to resolve. The
generated code is fast, but long winded and basically unnecessary since the only
requirement really is ordered object key/value fields that match the struct
specification.

Thus, to use this on other structs that will be JSON encoded, an example of
doing this can be seen here:

```go
// RelayLimits specifies the various restrictions and limitations that apply to
// interactions with a given relay.
type RelayLimits struct {
MaxMessageLength int  `json:"max_message_length,omitempty"`
MaxSubscriptions int  `json:"max_subscriptions,omitempty"`
MaxFilters       int  `json:"max_filters,omitempty"`
MaxLimit         int  `json:"max_limit,omitempty"`
MaxSubidLength   int  `json:"max_subid_length,omitempty"`
MaxEventTags     int  `json:"max_event_tags,omitempty"`
MaxContentLength int  `json:"max_content_length,omitempty"`
MinPowDifficulty int  `json:"min_pow_difficulty,omitempty"`
AuthRequired     bool `json:"auth_required"`
PaymentRequired  bool `json:"payment_required"`
RestrictedWrites bool `json:"restricted_writes"`
}

func (ri *RelayLimits) ToObject() (o object.T) {
return object.T{
{"max_message_length,omitempty", ri.MaxMessageLength},
{"max_subscriptions,omitempty", ri.MaxSubscriptions},
{"max_filters,omitempty", ri.MaxFilters},
{"max_limit,omitempty", ri.MaxLimit},
{"max_subid_length,omitempty", ri.MaxSubidLength},
{"max_event_tags,omitempty", ri.MaxEventTags},
{"max_content_length,omitempty", ri.MaxContentLength},
{"min_pow_difficulty,omitempty", ri.MinPowDifficulty},
{"auth_required", ri.AuthRequired},
{"payment_required", ri.PaymentRequired},
{"restricted_writes", ri.RestrictedWrites},
}
}
```

As you can see, the tag names in an `object.T` can have an appended `,omitempty`
and when t he `Buffer()` method, which writes the JSON efficiently into a bytes
buffer, encounters this in the tag name, it splices out the tag name on one
hand, and then checks for zero and nil value on the field and skips it if it
matches either condition.

Very likely it would be relatively simple to write a generator tool that
produces the second function automatically from the struct specification, but
with the use of multi-caret and copy/paste in an editor it is a very brief
exercise and due to being right beside it, should not be easily forgotten to
synchronise between the two. Such a generator will be left for future work.

## custom types

As can be seen in the Event example, there is two custom
types, `kind.T`, `timestamp.T` and `tags.T`. In order to integrate custom types
that may have incorrect naive interpretation by `encoding/json` such
as `time.Time` producing the default canonical time with timezone string, when
in general in JSON protocols these timestamps are converted to Unix timestamps
as decimal numbers, or as with the tags, that in Go the native default string
generator does not place commas between the field values, but only spaces, it is
essential where the output of the `Buffer()` and asssociated `Bytes()`
and `String()` methods yields the wrong data structure, that you create
custom `String()` methods for the type which produce the intended JSON
formatting.

Since JSON is basically the de facto standard internet text based data encoding
protocol, it is unlikely to be problematic for the stringers to yield JSON
compliant output in such cases as simple logging, since it's recognisable to
most programmers.

Even if JSON has stupid features like not allowing easily grown lists by
allowing final commas in lists eg: [1,2,3,] which make it easier for humans to
add entries to vertically separated, readable forms of the encoding, and the key
problem of there being no distinction between a structure and a key/value map,
it is what we have, and probably what most of us deserve, except us Go
programmers, who deserve people to build libraries like this one to eliminate
disadvantages that *normal*programmers don't understand... because they don't
know computer language theory or computer science, producing ever expanding
bloated abstractions that periodically result in inevitable, and avoidable
crises, if they would just listen to people like Ken Thompson, Rob Pike, and the
rest of the greats of CS language technology.

## TODO

This doesn't have the facility to embed an object inside an object currently,
because there wasn't a need for it yet. This is just a heads-up to let you know
this hasn't been done yet. Probably it can already be done but I can't say for
sure.