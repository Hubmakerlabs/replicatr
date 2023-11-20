# replicatr

nostr relay with modular storage and connectivity built on a publish-subscribe
model

## about

`replicatr` is a `nostr` relay written in pure Go, aimed at becoming a single,
modular, and extensible reference implementation of the `nostr` protocol as
described in the
nostr [NIP (nostr implementation possibilities) specification](https://github.com/nostr-protocol/nips).

In its initial form it will use a [badger](https://github.com/dgraph-io/badger)
data store, interface with
the [internet computer](https://internetcomputer.org/) database for out-of-band
replication and potentially ICP based relay subscription payments, and
implement a publish/subscribe system based on
the [Scionic Merkle DAG](https://github.com/HORNET-Storage/scionic-merkletree)
content addressing scheme, aggregating user events with associated media and
complex data types such as Git repositories and immutable filesystem tree
structured data, for its primary mechanism of event propagation.

### on-demand data distribution and replication

Social networks function as a generalised publish-subscribe distributed database
system.

In order to implement this, users are connected to one or a few nodes in the
network, a primary and secondaries that pick up the slack if the primary is
unresponsive.

Their messages are stored on their nodes, and other users who are subscribing to
their published events, their nodes set up subscriptions with the nodes the user
is
attached to and when new events are published, they are propagated to the
subscribing relays in order to deliver them to the users.

This model applies no matter whether all of the replicas are owned by one
organisation, who is distributing the data geographically for storage and
network efficiency, or if there is many organisations and individuals running
replicas of data on the network.

There is quite some challenges involved in engineering these systems to not
over-replicate data to the point that individual nodes are overly burdened with
data that is not actually used or propagated from them, and on the other side,
to prevent data from becoming unavailable or insufficiently replicated.

### economising replication costs

Thus, `replicatr` adds additional logic to the standard `nostr` relay model that
efficiently provides a compact distributed database index of users and their
aggregated event streams, allows relays to effectively collaborate to ensure
availability without excessively adding redundancy and burdening the network
with idle data.

The indexing is done using a new technique called "Scionic Merkle DAGs" (
SMD) https://github.com/HORNET-Storage/scionic-merkletree that enable relays to
quickly query each other for copies of data that their users are requesting or
are interlinking with their data (via reply threads and tagging) to efficiently
replicate and distribute the data of users published events while enabling users
to efficiently shard their data storage and minimise infrastructure costs and
propagation delays. This covers not just simple text based events but media
files and complex data structures such as Git repositories.

### extensible storage and network implementations

The standard connectivity protocol of `nostr` is websockets, and the standard
encoding is JSON. These were chosen as a baseline to enable the easy integration
of web browser javascript execution engines, which use these two technologies as
their primary methods of transferring data.

However, for bulk replication, and generally for scalability, JSON is a
terrible, extremely inefficient format with a complex syntax, and websockets are
excessively complex layering on top of HTTP and TCP, and for rapid
synchronisation between to nodes on a network, a UDP based, binary encoded
format is far more suitable. For this, any number of implementations could be
added, at minimum, QUIC and Protocol Buffers should be added, as these are
widely supported and more efficient.

Likewise, for cases of bulk synchronisation without a latency minimisation
requirement, the efficiently encoded form of the map of data objects associated
with user accounts, more complex, larger multiplexed storage schemes would be
more suitable than piecemeal delivery of individual events, and discovering the
missing pieces on either side of a connection efficiently so as to minimise
traffic and processing.

#### leveraging existing distributed data storage systems

In addition to aiming to provide a framework within `replicatr`to enable a
multiplicity of data storage and data replication systems, a key target in the
initial release of this project is to integrate it to one or more highly
consistent distributed database ledger protocols, aka "blockchains", both as
reliable replicas of the indexes of data, as well as directly storing the data,
where the protocol in question has this facility built into it.

This will beimplemented in the initial release of `replicatr` on
the `internet computer protocol`.

### multimedia distribution

In addition, with the assistance of the SMD data chunking/indexing system, it is
possible to not only distribute the data of simple text based events, but also
associated media.

Two main functionalities are targeted for this, one is the capture and caching
of data stored on referenced CDNs, and relays creating special event types that
aggregate these references so relays can distribute the data directly instead of
forcing the clients to provide access timing metadata to such CDNs, a serious
privacy risk and surveillance method (aka "web bugs") in addition to enabling
clients to publish the data directly to their relays.

In this way, we can have the data remain purely distributed across the relays as
their primary network location, and enabling users to aggregate their access
without leaking their access metadata to potentially malicious surveillance
operators.

As well as simple multimedia, it should be possible and simple for users to
publish complex data types, such as filesystem archives, and the more complex
branched merkle DAG based Git style filesystem with change history, to enable
also the distribution of software and source code with auditability via open
source, without centralised silos with the potential to suppress the
distribution of user data against the interests of the users and community at
large.

### conclusion

`replicatr` aims to build a resilient, efficiently redundant distribution system
for event publication and associated media, in a way that is loosely coupled,
extensible, and largely self-healing, that can be easily monetised directly via
subscription fees and per-access micropayment systems.

## structure of this repository

The pieces from which this repository is composed are taken from a scattered
collection of repositories mostly written
by [fiatjaf](https://github.com/fiatjaf) and gathered together to
become a single reference point for implementing `nostr` relays in pure Go.

### pkg/nostr

This directory contains a revised form of the content found
at [github.com/nbd-wtf/go-nostr](https://github.com/nbd-wtf/go-nostr) rewritten
to correct, idiomatic and properly documented Go code, for all things required
by `replicatr` to implement the `nostr` protocol.

