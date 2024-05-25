# Relay Clusters and Shared Storage

By enabling relays to share a `layer 2` event store, the relay functions as a fast cache that can deliver freshly
published events quickly to users, while enabling multiple relays to deliver this data, thus a relay cluster can cope
with very large numbers of users, while keeping users data connected between the relays, serving to distribute the load
of the work of serving and storing events.

The individual relays have a storage management feature that allows the limitation of storage usage to the confines of
what the relay operator has provisioned for this purpose, with configurable parameters of garbage collection frequency,
and the high and low water marks, the former is the trigger size, and the latter is the target that each garbage
collection run will cut the storage usage back down to. Events have last access time records and the oldest ones are
pruned first.

The storage management can be used by itself, without a `layer 2` and for this case the stale events are just completely
removed.

If used with a `layer 2` shared store like the Internet Computer Protocol canister, instead of deleting the whole event
data, only the event data itself is removed, and the search indexes remain, and are able to fill up the space between
the high water mark and the database size limit. These are also pruned by age once the total size of the indexes exceeds
the headroom between the high water mark and the limit.
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
