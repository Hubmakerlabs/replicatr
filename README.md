## Replicatr
![logo](doc/logo.png)

replicatr is a relay for the [nostr protocol](https://github.com/nostr-protocol/nostr)

* Supports most applicable NIPs: 1, 2, 4, 9, 11, 12, 15, 16, 20, 22, 28, 33, 40, 42
* Websocket compression: permessage-deflate with optional sliding window, when supported by clients
* Extremely configurable making it seamlessly customizable across several parameters for relay operators 
* No external database required: All data is stored first locally on the filesystem in BadgerDB and optionally on the Internet Computer for inter-relay synchronization. 
* The local badgerDB is additionally equipped with a nostr-specific, highly configurable garbage collection scheme and a nostr-specific prefix-indexing scheme for seamless data mangement and rapid querying
* Supports optionally mandating nip-42 authorization upon initial connection for enhanced security
* [Internet Computer](https://internetcomputer.org/docs/current/home)-integration for efficient syncing with remote relays

## Syncing

The most original feature of replicatr is it's  [Internet Computer](https://internetcomputer.org/docs/current/home) integration allowing for quick and seamless inter-relay synchronization. This is achieved by defining relay clusters, an interconnected group of relays that're given authorization by a replicatr canister owner to utilize the canister's synchronization tooling to achieve consistency across the cluster.

Click here** to learn more about the problem this solves.

Click here** to learn more about the synchronization architecture.

## Usage
### Setup

 1. To setup an Owner relay(and start your own cluster):
	- [ ] Clone the repo and ensure golang is installed.
	- [ ] Ensure [dfx](https://internetcomputer.org/docs/current/developer-docs/getting-started/install/) is installed in the repo root directory with a nonzero [cycle balance](https://support.dfinity.org/hc/en-us/articles/5946641657108-What-is-a-cycles-wallet).
	- [ ] From the root directory, run the initialization script:\
	`chmod +x pkg/ic/setup/owner.sh
	./pkg/ic/setup/owner.sh`\
	(This will initialize your relay and deploy a replicatr canister on the Internet Computer with your relay as the specified owner.)
	     
	     
 2. To setup as a Minion/Secondary Owner  relay(and join a preexisting cluster):
	 - [ ] Identify the a relay cluster you would like to join and ask the owner for their canister-id and if you can join.
	 - [ ] Clone the repo and ensure golang is installed
	 - [ ] Run the following command from the root directory to initialize the relay with the previously obtained canister-id:\
	 `go run .  initcfg -I <canister-id>`
	 - [ ] Run the following command to obtain your canister-facing relay pubkey:\
	 `go run . pubkey`
	 - [ ] Send the resulting pubkey to the canister owner and wait for them to grant you user/owner level access

To learn more about canister permissions, click here**.
