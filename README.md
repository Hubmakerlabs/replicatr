# replicatr

nostr relay with modular storage and connectivity

## about

`replicatr` is a `nostr` relay written in pure Go, aimed at becoming a single,
modular, and extensible reference implementation of the `nostr` protocol as
described in the
nostr [NIP (nostr implementation possibilities) specification](https://github.com/nostr-protocol/nips).

It will use a [badger](https://github.com/dgraph-io/badger)
data store for local caching, and interface with
the [internet computer](https://internetcomputer.org/) for storage of all 
event types except ephemeral and private events.
