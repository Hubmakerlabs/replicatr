# Replicatr

![logo](doc/logo.png)

`replicatr` is a relay for the [nostr protocol](https://github.com/nostr-protocol/nostr)

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
interconnected group of relays that're given authorization by a replicatr canister owner to utilize the canister's
synchronization tooling to achieve consistency across the cluster.

> [Click here](docs/cluster.md) to learn more about the problem this solves.

> [Click here](docs/sync.md) to learn more about the synchronization architecture.

## Usage

### Setup

Works with Linux, MacOS, and WSL2

### Install Go

Go 1.17+ is recommended - unlike most other languages, the forward compatibility
guarantee is ironclad, so go to [https://go.dev/dl/](https://go.dev/dl/) and
pick the latest one (1.22.3 at time of writing), "copy link location" on the
relevant version (linux x86-64 in this example, which applies to Linux and WSL, for Mac [see here](https://go.dev/dl/) -
not tested for BSDs or Windows but should work).

```bash
cd
mkdir bin 
wget https://go.dev/dl/go1.22.3.linux-amd64.tar.gz
tar xvf go1.18.linux-amd64.tar.gz
```

Using your favourite editor, open up `~/.bashrc` - or just

```bash
nano ~/.bashrc
```

and put the following lines at the end

```bash
export GOBIN=$HOME/bin
export GOPATH=$HOME
export GOROOT=$GOPATH/go
export PATH=$HOME/go/bin:$HOME/.local/bin:$GOBIN:$PATH
``` 

save and close, and `ctrl-d` to kill the terminal session, and start a new one.

This also creates a proper place where `go install` will put produced binaries.

### Building and Running

You can run the relay directly from the root of the repository:

```bash
go run . <flags> <args>
```

Or you can build it and place it in the location `GOBIN` as defined above:

```bash
go install
```

Add flags to configure the relay as needed or run without any flags to use defaults:

```
Usage: replicatr [--listen LISTEN] [--eventstore EVENTSTORE] [--canisteraddr CANISTERADDR] 
    [--canisterid CANISTERID] [--profile PROFILE] [--name NAME] [--description DESCRIPTION] 
    [--pubkey PUBKEY] [--contact CONTACT] [--icon ICON] [--auth] [--public] [--owner OWNER] 
    [--seckey SECKEY] [--whitelist WHITELIST] [--allow ALLOW] [--sizelimit SIZELIMIT] 
    [--lowwater LOWWATER] [--highwater HIGHWATER] [--gcfreq GCFREQ] [--maxprocs MAXPROCS] 
    [--loglevel LOGLEVEL] [--pprof] [--gcratio GCRATIO] [--memlimit MEMLIMIT] 
    [--pollfrequency POLLFREQUENCY] [--polloverlap POLLOVERLAP] <command> [<args>]

Options:
  --listen LISTEN, -l LISTEN
                         network address to listen on
  --eventstore EVENTSTORE, -e EVENTSTORE
                         select event store backend [ic,badger,iconly]
  --canisteraddr CANISTERADDR, -C CANISTERADDR
                         IC canister address to use (for local, use http://127.0.0.1:<port number>)
  --canisterid CANISTERID, -I CANISTERID
                         IC canister ID to use
  --profile PROFILE, -p PROFILE
                         profile name to use for storage [default: replicatr]
  --name NAME, -n NAME   name of relay for NIP-11
  --description DESCRIPTION, -d DESCRIPTION
                         description of relay for NIP-11
  --pubkey PUBKEY        public key of relay operator
  --contact CONTACT, -c CONTACT
                         non-nostr relay operator contact details
  --icon ICON, -i ICON   icon to show on relay information pages
  --auth, -a             NIP-42 authentication required for all access
  --public               allow public read access to users not on ACL
  --owner OWNER, -o OWNER
                         specify public keys of users with owner level permissions on relay
  --seckey SECKEY, -s SECKEY
                         identity key of relay, used to sign 30066 and 30166 events and for message control interface
  --whitelist WHITELIST, -w WHITELIST
                         IP addresses that are only allowed to access
  --allow ALLOW, -A ALLOW
                         IP addresses that are always allowed to access
  --sizelimit SIZELIMIT, -S SIZELIMIT
                         set the maximum size of the badger event store in bytes
  --lowwater LOWWATER, -L LOWWATER
                         set target percentage for database size during garbage collection
  --highwater HIGHWATER, -H HIGHWATER
                         set garbage collection trigger percentage for database size during garbage collection
  --gcfreq GCFREQ, -G GCFREQ
                         frequency in seconds to check if database needs garbage collection
  --maxprocs MAXPROCS    maximum number of goroutines to use
  --loglevel LOGLEVEL    set log level [off,fatal,error,warn,info,debug,trace] (can also use GODEBUG environment variable)
  --pprof                enable CPU and memory profiling
  --gcratio GCRATIO      set GC percentage for triggering GC sweeps
  --memlimit MEMLIMIT    set memory limit on process to constrain memory usage
  --pollfrequency POLLFREQUENCY
                         if a level 2 event store is enabled how often it polls
  --polloverlap POLLOVERLAP
                         if a level 2 event store is enabled, multiple of poll freq overlap to account for latency
  --help, -h             display this help and exit

Commands:
  initcfg                initialize relay configuration files
  export                 export database as line structured JSON
  import                 import data from line structured JSON
  pubkey                 print relay canister public key
  addrelay               add a relay to the cluster
  removerelay            remove a relay from the cluster
  getpermission          get permission of a relay
  wipebdb                empties local badger database (bdb)

```

By default, `replicatr` creates a profile folder in `$HOME/replicatr` use the `-p` or `--profile` folder to change the
name, the location is not configurable, however. Add a dot prefix to have it hidden from regular directory listings eg
`-p .localcache`

Inside this folder will be stored two configuration files, `config.json` and `info.json`. The first contains persistent
settings based on the CLI arguments shown above, and the second contains a
[NIP-11](https://github.com/nostr-protocol/nips/blob/master/11.md) Relay Information Document. Any relevant parameters
set on the command line will override those found in these two files (such as `--auth`) for the duration of the run.

#### Compiling

#### To setup an Owner relay (and start your own cluster):

1. Clone the repo and ensure golang (v1.20+) is installed.
2. Ensure [dfx](https://internetcomputer.org/docs/current/developer-docs/getting-started/install/) is installed in the
   repo root directory with a
   nonzero [cycle balance](https://support.dfinity.org/hc/en-us/articles/5946641657108-What-is-a-cycles-wallet).
3. From the root directory, run the initialization script:

```
chmod +x pkg/ic/setup/owner.sh
./pkg/ic/setup/owner.sh
```

> This will initialize your relay and deploy a replicatr canister on the Internet Computer with your relay as the
> specified owner.

#### To setup as a Minion/Secondary-Owner  relay (and join a preexisting cluster):

1. Identify the a relay cluster you would like to join and ask the owner for their canister-id and if you can join.
2. Clone the repo and ensure golang (v1.20+) is installed
3. Run the following command from the root directory to initialize the relay with the previously obtained canister-id:\
   `go run . initcfg -I <canister-id>`
4. Run the following command to obtain your canister-facing relay pubkey:\
   `go run . pubkey`
5. Send the resulting pubkey to the canister owner and wait for them to grant you user/owner level access

> To learn more about canister permissions, [click here](docs/canister.md).

