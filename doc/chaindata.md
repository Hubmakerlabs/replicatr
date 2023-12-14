# Blockchain/Nostr API

## Introduction

Messaging patterns are fundamental to how distributed systems are constructed. In the case of Internet Computer Protocol and Nostr, there is two very different communication patterns going on:

- nostr: publish subscribe - highly parallel and distributed, with some data more widely demanded than other

- IC: Inside subnets, data is replicated within 3-5 seconds across many nodes, the pattern enables parallel read but writing is serialized into blocks. 

> Communication between subnets is slower and more expensive, due to requirements for handshaking, authentication and synchronization, the 3-5 seconds write latency) and competing uses of the cluster's connectivity (as would be the case if the IC were running websockets for nostr).
>
> Nostr relays, on the other hand, maintain open websockets between each other constantly and stream data without warning, no sync, handshake or authentication required.

As such, there is some overlap that you can see in the second clause of the nostr point. Blockchains have also been referred to as ["replicated state machines"](https://en.wikipedia.org/wiki/State_machine_replication) meaning that their primary purpose and messaging pattern is specifically designed to make reading from any node in the network produce the same data, within a narrow time window called finality (consistency).

Thus, similar to flash storage, the writing is expensive, and the reading is cheap. Writing is slow, reading is fast. In the disk storage analogy, spinning disks have an equal time between reading and writing, their weakness is seeking the data, because of the mechanical data reader.

This analogy holds pretty well to compare blockchain synchronisation versus publish/subscribe data distribution as can be seen as a contrast between Internet Computer Protocol and Nostr. Data is not replicated completely, because this is impractical in terms of data volume and message complexity, and unnecessary because demand for content is widely varying across the userbase.

The distinction between the two storage systems breaks down in the way that reading from local stores is fast, but finding the place where data not already replicated is slow, but the overall point that read and write are equal in focus versus a read-oriented optimization of blockchain still applies.

What we are aiming to achieve with `replicatr` is to put that part of the Nostr data set that needs to be widely replicated and doesn't have a high volume of changes, or doesn't require updating of old data (append only) onto a blockchain so that relays connected to the same blockchain back end do not have to specifically request this data from each other anymore, and it is frequently requested, **it constitutes one of the biggest bottlenecks of the protocol.**

*By using a blockchain for this type of data, we improve the performance of the relays that use it, as well as build a bridge from the blockchain world to the Nostr world that gives you the best of both worlds.*

## Data Types and Queries