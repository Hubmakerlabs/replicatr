# replicatr

nostr relay with modular storage and connectivity

## about

`replicatr` is a `nostr` relay written in pure Go, aimed at becoming a single,
modular, and extensible reference implementation of the `nostr` protocol as
described in the
nostr [NIP (nostr implementation possibilities) specification](https://github.com/nostr-protocol/nips).

It will use a [badger](https://github.com/dgraph-io/badger)
data store, interface with
the [internet computer](https://internetcomputer.org/) interface for often read less often written event types (all except ephemeral, private and bulky data content).
