# Replicatr

![logo](doc/logo.png)

`replicatr` is a relay for the [nostr protocol](https://github.com/nostr-protocol/nostr) with support for `layer 2` event stores, such as the Internet 
Computer Protocol event store canister.

* Supports most applicable NIPs: 1, 2, 4, 9, 11, 12, 15, 16, 20, 22, 28, 33, 40, 42
* Websocket compression: permessage-deflate with optional sliding window, when supported by clients
* Extremely configurable making it seamlessly customizable across several parameters for relay operators
* No external database required: All data is stored first locally on the filesystem in BadgerDB and optionally on the
  Internet Computer for inter-relay synchronization.
* The local badgerDB is additionally equipped with a nostr-specific, highly configurable garbage collection scheme and a
  nostr-specific prefix-indexing scheme for seamless data mangement and rapid querying
* Supports optionally mandating nip-42 authorization upon initial connection for enhanced security
* [Internet Computer](https://internetcomputer.org/docs/current/home)-integration for efficient syncing with remote
  relays

## Syncing

The most original feature of replicatr is it's  [Internet Computer](https://internetcomputer.org/docs/current/home)
integration allowing for quick and seamless inter-relay synchronization. This is achieved by defining relay clusters, an
interconnected group of relays that're given authorization by a replicatr [canister](https://internetcomputer.org/docs/current/concepts/canisters-code) owner to utilize the canister's
synchronization tooling to achieve consistency across the cluster.

> [Click here](doc/cluster.md) to learn more about the problem this solves.

> [Click here](doc/sync.md) to learn more about the synchronization architecture.

## Usage

### Setup

Works with Linux, MacOS, and WSL2

#### Install Go and Clone Repo

Go 1.2+ is recommended - Click [here](doc/golang.md) for installation instructions and specifications.

Then, run to the following to clone the repo:

```bash
git clone https://github.com/Hubmakerlabs/replicatr.git && cd replicatr
```


#### To setup an Owner relay (and start your own cluster):


1. Ensure [dfx](https://internetcomputer.org/docs/current/developer-docs/getting-started/install/) and all corresponding dependencies are installed in the
   repo root directory. Ensure a valid [dfx identity](https://internetcomputer.org/docs/current/developer-docs/developer-tools/cli-tools/cli-reference/dfx-identity) with an [initialized wallet](https://internetcomputer.org/docs/current/developer-docs/developer-tools/cli-tools/cli-reference/dfx-quickstart) is created and is being used.
2. Use [NNS](https://nns.ic0.app/) to [create a canister](https://internetcomputer.org/docs/current/developer-docs/daos/nns/nns-app-quickstart) and [top it up](https://internetcomputer.org/docs/current/developer-docs/smart-contracts/topping-up/topping-up-canister) with at least half an ICP worth of cycles (or more depending on your intended bandwidth usage).
3. From the root directory, run the initialization script:

```bash
chmod +x pkg/ic/setup/owner.sh
./pkg/ic/setup/owner.sh
```
Input the canister-id for the previously created canister when prompted:

```bash
Please enter the canister ID: <canister-id>
```

> This will initialize your relay and deploy a replicatr canister on the Internet Computer with your relay as the
> specified owner.

#### To setup as a Minion/Secondary-Owner  relay (and join a preexisting cluster):

1. Identify the a relay cluster you would like to join and ask the owner for their canister-id and if you can join.
2. Run the following command from the root directory to initialize the relay with the previously obtained canister-id:

   ```bash
   go run . initcfg -I <canister-id>
   ```
   
5. Run the following command to obtain your canister-facing relay pubkey:
   ```bash
   go run . pubkey
   ```
   
7. Send the resulting pubkey to the canister owner and wait for them to grant you user/owner level access

> To learn more about canister permissions, [click here](doc/canister.md).

### Building and Running

You can run the relay directly from the root of the repository:

```bash
go run . <flags> <args>
```

Or you can build it and place it in the location `GOBIN` as defined [here](doc/golang.md):

```bash
go install
```

> add flags to configure the relay as needed or run without any flags to use defaults. Click [here](pkg/config/base/README.md) to view customizable parameters, configuration, and subcommand details



