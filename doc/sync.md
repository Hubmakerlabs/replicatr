# Synchronisation of Relay Data

With multiple relays sharing a network connected event store, such as the Internet Computer implementation found in this
repository, the relays check every 5 seconds (a configurable frequency) for the last 15 seconds of events that have been
added to the `layer 2` event store.

This enables the several relays to be continually updated so messages and posts on one can be interacted with by users
connected to other relays in the cluster. This synchronization is fast enough to carry instant messages as well as
general forum posts.

Potentially in future the canisters could keep request filters open and when new events arrive, actively push the events
back to the other relays in the cluster, however this is more complex and raises the processing and memory requirements,
and may not reduce latency by very much, thus it is not implemented in the initial version.