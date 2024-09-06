# Internet Computer Protocol/Nostr API

Specification of message types and data structures that will be passed to and fro between the ICP canisters and the `replicatr` relay.

## Introduction

Messaging patterns are fundamental to how distributed systems are constructed. In the case of Internet Computer Protocol and Nostr, there is two very different communication patterns going on:

- nostr: publish subscribe - highly parallel and distributed, with some data more widely demanded than other

- IC: Inside subnets, data is replicated within 3-5 seconds across many nodes, the pattern enables parallel read but writing is serialized into blocks. 

> Communication between subnets is slower and more expensive, due to requirements for handshaking, authentication and synchronization, the 3-5 seconds write latency) and competing uses of the cluster's connectivity (as would be the case if the IC were running websockets for nostr).
>
> Nostr relays, on the other hand, maintain open websockets between each other constantly and stream data without warning, no sync, handshake or authentication required.

As such, there is some overlap that you can see in the second clause of the nostr point. Blockchains have also been referred to as ["replicated state machines"](https://en.wikipedia.org/wiki/State_machine_replication) meaning that their primary purpose and messaging pattern is specifically designed to make reading from any node in the network produce the same data, within a narrow time window called finality (consistency).

### Blockchains, Publish Subscribe and Disk Storage Analogy

Thus, similar to flash storage, the writing is expensive, and the reading is cheap. Writing is slow, reading is fast. In the disk storage analogy, spinning disks have an equal time between reading and writing, their weakness is seeking the data, because of the mechanical data reader.

This analogy holds pretty well to compare blockchain synchronisation versus publish/subscribe data distribution as can be seen as a contrast between Internet Computer Protocol and Nostr. On Nostr, data is not replicated completely, because this is impractical in terms of data volume and message complexity, and unnecessary because demand for content is widely varying across the userbase.

Very likely, optimization strategies for data locality that work on spinning disks can be transposed to create optimistic early propagation patterns for data that tend to fill the caches where their likely highest locality exists, similar in the linear sense of seek time but different in that it's 2 dimensional and not necessarily geographical but based on user association with relays. Using a blockchain like IC is an easy optimization for data that inherently has the broadest demand distribution.

What we are aiming to achieve with `replicatr` is to put that part of the Nostr data set that needs to be widely replicated and doesn't have a high volume of changes, or doesn't require updating of old data (append only) onto a blockchain so that relays connected to the same blockchain back end do not have to specifically request this data from each other anymore, and it is frequently requested, **it constitutes one of the biggest bottlenecks of the protocol.**

*By using a blockchain for this type of data, we improve the performance of the relays that use it, as well as build a bridge from the blockchain world to the Nostr world that gives you the best of both worlds.*

## API Calls

For the most part, as with the disk storage analogy, most of what we implement is simply asking for data, and publishing data.

For these, there is the standard Nostr messages, the Event Envelope, for publication, and the Request Envelope, which contains a collection of matching criteria for a search. 

### Publishing Events

In the initial implementation we strictly assign a list of event types that are stored in complete form on the IC, these are events that have small size and wide demand, and those that it only stores metadata about, as a means to accelerating the location of relays holding the referred to events.

The data that is most likely to be required for simply locating the full content and signed versions is the post ID (hash), the datestamp on the event, and the kind of event. Some event types we may also want to store some or all of the tags found on the event tags, if the data is relatively small, easily compressed or turned into internal database references instead of full sized nostr network event identites.

### Retrieving Events

The criteria are embedded in a data structure called a Filter, of which all fields are optional and can be combined freely:

#### Filters

Filters are a message type that appears inside a REQ envelope. The exact form to use for IC canister requests may differ from this because many of the events that IC will handle are replaceable type events that only the newest one is kept. The majority of the rest of the data only keeps reference information such as ID, pubkey, timestamp, and lists of relays known to have this data. The search field is probably irrelevant as it is for searching for matches in content. Possibly in the future there may be keyword indexes that allow some of this location to be stored on the IC.

```json
{
  "ids": [<a list of event ids>, ...],
  "authors": [<lowercase pubkey, the pubkey of an event must be one of these>, ...],
  "kinds": [<kind (event type) number>, ...]
  "#<single-letter (a-zA-Z)>": [<tags, for #e — a list of event ids, for #p — a list of event pubkeys etc>, ...],
  "since": <an integer unix timestamp in seconds, events must be newer than this to pass>,
  "until": <an integer unix timestamp in seconds, events must be older than this to pass>,
  "limit": <maximum number of events relays SHOULD return in the initial query>,
  "search": "search string"
}
```

### Authentication and Permissions

The main and specific types of API calls needed to integrate with IC are authentication. In order to control unbounded traffic (spam) from occurring, IC canisters have a list of keys with the permissions to read and write data to the IC and have requests fulfilled (including simple data fetches).

In general, relays must have read and write access, and external, web3 apps that interface with the IC canister will have at least read access, but may have write access as well. This is entirely dependent on the design of the applications, but in principle, there can be the use of the IC as a conduit through which web3 apps, that may be running on other canisters, or even other chains over network bridging interfaces, can add data that can propagate to the relays and enter the Nostr network. For the most part this would be web3 nostr clients.

Thus, the depth of the functionality of the IC canister must be extensible to eventually encompass full processing of at least Event and Request Envelopes, but the IC/Web3 side of the API is more of a concern for dApp developers as opposed to Nostr clients, which are our focus as the main input/output path for events and searches.

Nostr implementations largely don't have a focus on optimizing data distribution for more than simple text events, but adding blockchain back end connectivity for propagating suitable types of data is a step in this direction towards making the network more heteregenous and relays more purpose-built.

### Search Optimization - Merkle Trees and Bloom Filters

For improving the ability to find data, various indexing and cataloging strategies can be used. Merkle Trees, Merkle Directed Acyclic Graphs, and other similar structures can be generated and used to speed up seek time.

In addition, for events tied to single pubkeys, we can generate Bloom Filters, which are a probabalistic data structure that can quickly determine if a candidate event identity, for instance, is associated with a public key, and by this create compact recognisers that can quickly check from a request whether an event's references connect to a user, such as thread replies.

## Data Types Suited to Blockchain Databases

The native encoding of Nostr is JSON, but this is only mandatory in data types that have canonical forms that are hashed and signed. In storage and on the wire, there can be other encodings such as BSON, MessagePack, Protobuf, CBOR, so long as the encoding can be turned into JSON for messaging relays that do not implement the alternative encodings.

The following list is of the data types that will be stored on the IC, as a reference for those building the IC canister that will interface with the `replicatr` nostr relay.

Events are selectively stored whole by the IC, based on the criteria of their size and the directions from which demand for them is likely to come. Profile data and community and chat events that do not have their own body of content, or if they are wrapped inside another event, this can be identified by the kind number of the event.

Conversely, messages that the relay can rely on to be stored on the IC are not kept in the longer term storage area but in the smaller cache buffer where data that did not originate from events published to the relay, or are the concern of the IC, are stored and eventually evicted once their last access time is in the oldest segment of the cache.

Events that are not stored on the IC only have the ID, timestamp, pubkey of the author (compactly stored as a fingerprint alongside an index) and depending on the kind, maybe the tags. The content and signature are elided as they are not useful to the filter requests. The relays that have copies of the full event are noted alongside these details, and these relays will then be queried for the full content to enable such things as full text searches or delivering the event to a client.

### Event

Note that for most events, signature, content and tags should not be stored, primarily the references, the post ID, the poster pubkey - which likely will be a shorter fingerprint such as 16 bytes rather than 32, with a separate index, as there is only a limited number of accounts that will be created. Relays will store the complete events compactly (as binary) in their databases and use snappy compression to reduce the size of the content field.

```json
{   
    "id": <32-bytes lowercase hex-encoded sha256 of the serialized event data>,
    "pubkey": <32-bytes lowercase hex-encoded public key of the event creator>,
    "created_at": <unix timestamp in seconds>,
    "kind": <integer between 0 and 65535>,
    "tags": [ [  <arbitrary string>, <another string> ... ], ... ],
    "content": <arbitrary string, control characters and non printables escaped using \uXXXX and \n\t\r etc>,
    "sig": <64-bytes lowercase hex of the signature of the sha256 hash of the serialized event data, which is the "id" field>
}
```

### Canonical format of event for ID hash generation

Note here shown squashed together without whitespace, as it must be for the generation of the ID via SHA256 hash (`id` field) that signatures are generated on:

```json
[0,"<pubkey, as a lowercase hex string>",<created_at as a decimal integer of unix timestamp>,<kind as a number>,<tags as an array of arrays of non-null strings>,"<content, as a string>"]
```
### Tags

Tags are lists of lists of strings, usually starting with a field that is a single letter, but these strings can be anything at all. Below is some examples of tags that appear in events. Note that tags can also appear in request REQ message envelopes.


```json
{
  ...,
  "tags": [
    ["e", "5c83da77af1dec6d7289834998ad7aafbd9e2191396d75ec3cc27f5a77226f36", "wss://nostr.example.com"],
    ["p", "f7234bd4c1394dda46d09f35bd384dd30cc552ad5541990f98844fb06676e9ca"],
    ["a", "30023:f7234bd4c1394dda46d09f35bd384dd30cc552ad5541990f98844fb06676e9ca:abcd", "wss://nostr.example.com"],
    ["alt", "reply"],
    ...
  ],
  ...
}
```

### Follow Lists

The following are essentially specifics of how the event will be formed, `id`, `pubkey` , `created_at` and `sig` will also appear in these events.

```json
{
  "kind": 3,
  "tags": [
    ["p", "91cf9..4e5ca", "wss://alicerelay.com/", "alice"],
    ["p", "14aeb..8dad4", "wss://bobrelay.com/nostr", "bob"],
    ["p", "612ae..e610f", "ws://carolrelay.com/ws", "carol"]
  ],
  "content": "",
  ...other fields
}
```

### Public Chat Channel Messages

The following are specifics for how the public channel messages will be formed, again as with follow lists and most others, these are extra and distinctive parts of the standard `event` type described above.

This does not include chat text posts, only events that modify the channels and visibility of messages and users, which may persist indefinitely and need to be accessed from other relays quickly.

#### Create Channel

```json
{
  "content": "{\"name\": \"Demo Channel\", \"about\": \"A test channel.\", \"picture\": \"https://placekitten.com/200/200\"}",
  ...
}
```

#### Set Channel Metadata

```json
{
  "content": "{\"name\": \"Updated Demo Channel\", \"about\": \"Updating a test channel.\", \"picture\": \"https://placekitten.com/201/201\"}",
  "tags": [["e", <channel_create_event_id>, <relay-url>]],
  ...
}
```

#### Hide Message

```json
{
  "content": "{\"reason\": \"Dick pic\"}",
  "tags": [["e", <kind_42_event_id>]],
  ...
}
```

#### Mute User

```json
{
  "content": "{\"reason\": \"Posting dick pics\"}",
  "tags": [["p", <pubkey>]],
  ...
}
```

#### User Status

```json
{
  "kind": 30315,
  "content": "Sign up for nostrasia!",
  "tags": [
    ["d", "general"],
    ["r", "https://nostr.world"]
  ],
}
```

```json
{
  "kind": 30315,
  "content": "Intergalatic - Beastie Boys",
  "tags": [
    ["d", "music"],
    ["r", "spotify:search:Intergalatic%20-%20Beastie%20Boys"],
    ["expiration", "1692845589"]
  ],
}
```

### Lists and Sets

Lists are lists of things for various purposes, such as mute lists, follows, blocked relays, preferred relays, bookmarks, communities, and so on.. These are long lived and should be widely accessible, similar to the other event types, and thus also fit the requirements for IC stored events.

These event types are primarily encoded in tags, so the same principles apply - the IC data is only sent to it by authorized relays so the bulky signatures can largely be omitted as they aren't searchable anyway.

### Communities

These are membership-oriented groups where there are whitelisted members, blacklisted (banned) members, moderators, community metadata, and a collection of events tied to the community, and post moderation. These are examples of the several data types that appear in this type of event.

##### Community Creation

```json
{
  "created_at": <Unix timestamp in seconds>,
  "kind": 34550,
  "tags": [
    ["d", "<community-d-identifier>"],
    ["description", "<Community description>"],
    ["image", "<Community image url>", "<Width>x<Height>"],

    //.. other tags relevant to defining the community

    // moderators
    ["p", "<32-bytes hex of a pubkey1>", "<optional recommended relay URL>", "moderator"],
    ["p", "<32-bytes hex of a pubkey2>", "<optional recommended relay URL>", "moderator"],
    ["p", "<32-bytes hex of a pubkey3>", "<optional recommended relay URL>", "moderator"],

    // relays used by the community (w/optional marker)
    ["relay", "<relay hosting author kind 0>", "author"],
    ["relay", "<relay where to send and receive requests>", "requests"],
    ["relay", "<relay where to send and receive approvals>", "approvals"],
    ["relay", "<relay where to post requests to and fetch approvals from>"]
  ],
  ...
}
```

##### Post Request

```json
{
  "kind": 1,
  "tags": [
    ["a", "34550:<community eventenvelope author pubkey>:<community-d-identifier>", "<optional-relay-url>"],
  ],
  "content": "hello world",
  ...
}
```

#### Post Approval

```json
{
  "pubkey": "<32-bytes lowercase hex-encoded public key of the eventenvelope creator>",
  "kind": 4550,
  "tags": [
    ["a", "34550:<eventenvelope-author-pubkey>:<community-d-identifier>", "<optional-relay-url>"],
    ["e", "<post-id>", "<optional-relay-url>"],
    ["p", "<port-author-pubkey>", "<optional-relay-url>"],
    ["k", "<post-request-kind>"]
  ],
  "content": "<the full approved eventenvelope, JSON-encoded>",
  ...
}
```

#### Filtering for Approved Posts

```json
[
  "REQ",
  "_",
  {
    "authors": ["<owner-pubkey>", "<moderator1-pubkey>", "<moderator2-pubkey>", "<moderator3-pubkey>", ...],
    "kinds": [4550],
    "#a": ["34550:<Community eventenvelope author pubkey>:<d-identifier of the community>"],
  }
]
```



