# Replicatr

![logo](doc/logo.png)

`replicatr` is a relay for the [nostr protocol](https://github.com/nostr-protocol/nostr) with support for `layer 2` event stores, such as the Internet 
Computer Protocol event store canister.

* Supports most applicable NIPs: 1, 2, 4, 9, 11, 12, 15, 16, 20, 22, 28, 33, 40, 42
* Websocket compression: permessage-deflate with optional sliding window, when supported by clients
* Extremely configurable making it seamlessly customizable across several parameters for relay operators
* No external database required: All data is stored first locally on the filesystem in BadgerDB and optionally on the
  Internet Computer for inter-relay synchronization.
* The local badgerDB has been additionally equipped with a nostr-specific, highly configurable garbage collection scheme and a
  nostr-specific prefix-indexing scheme for seamless data mangement and rapid querying
* Supports optionally mandating nip-42 authorization upon initial connection for enhanced security
* [Internet Computer](https://internetcomputer.org/docs/current/home)-integration for efficient syncing with remote
  relays

## Syncing

The most original feature of replicatr is it's  [Internet Computer](https://internetcomputer.org/docs/current/home)
integration allowing for quick and seamless inter-relay synchronization. This is achieved by defining relay clusters, an
interconnected group of relays that're given authorization by a replicatr [canister](https://internetcomputer.org/docs/current/concepts/canisters-code) owner to utilize the canister's
synchronization tooling to achieve consistency across the cluster.

> [Click here](doc/problem.md) to learn more about the nostrific problem this addresses.

> [Click here](doc/arch.md) to learn more about the synchronization architecture.

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

## Additional Features

|Package Name|Links |Description|
|-----------------|-------|-----|
|`testr`          |       |     |
|`blowr`|[![README](https://img.shields.io/badge/-README-green)](cmd/blower/README.md)|CLI tool that facilitates the uploading of Nostr events from a JSONL file to a specified Nostr relay|
|`loggr`|[![README](https://img.shields.io/badge/-README-green)](doc/logger.md) [![DOC](https://img.shields.io/badge/-DOC-blue)](https://pkg.go.dev/mleku.dev/git/slog@v1.0.16)|highly-informative, configurable logger to monitor relay activity|
|`agent`| [![DOC](https://img.shields.io/badge/-DOC-blue)](https://pkg.go.dev/github.com/Hubmakerlabs/replicatr/pkg/ic/agent)|IC-tooling for Nostr data and managing Nostr relays' canister access|



## Credits

This project would not be possible without the significant contributions and support from the following organizations and projects:

1. **DFINITY** - For funding our project and enabling us to build on their cutting-edge [blockchain technology](https://internetcomputer.org/docs/current/home).
2. **[Hubmaker Labs](https://github.com/Hubmakerlabs)** - For their funding and ongoing support throughout the development process.
3. **nbd-wtf** - We forked their [`go-nostr`](https://github.com/nbd-wtf/go-nostr) package, which forms a substantial part of our Nostr tooling.
4. **fiatjaf** - We based our project on his [`khatru`](https://github.com/fiatjaf/khatru) relay, using it as the foundational structure for our development.
5. **Aviate Labs** - Their [`agent-go`](https://github.com/aviate-labs/agent-go) tooling has been instrumental in facilitating our interaction with Internet Computer canisters.

We extend our deepest gratitude to all our contributors and supporters, as their efforts and resources have been vital to the success of this project.




