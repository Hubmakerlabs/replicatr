

<p align="center">
  <img src="doc/logo.png" />
</p>

replicatr is a relay for the [nostr protocol](https://github.com/nostr-protocol/nostr)

* Supports most applicable NIPs: 1, 2, 4, 9, 11, 12, 15, 16, 20, 22, 28, 33, 40, 42
* Websocket compression: permessage-deflate with optional sliding window, when supported by clients
* Extremely configurable making it seamlessly customizable across several parameters for relay operators 
* No external database required: All data is stored first locally on the filesystem in BadgerDB and optionally on the Internet Computer for inter-relay synchronization. 
* The local badgerDB is additionally equipped with a nostr-specific configurable garbage collection scheme and prefix indexing for seamless data mangement and rapid querying
* Supports optionally mandating nip-42 authorization upon initial connection for enhanced security
* [Internet Computer](https://internetcomputer.org/docs/current/home)-integration for efficient syncing with remote relays
