# Configuration Parameters for `replicatr`

This document provides detailed explanations of the configurable parameters and subcommands for `replicatr`. These parameters can be set when running the relay, allowing for customization according to the operational needs.

## General Usage

```bash
replicatr [options] <command> [<args>]
```

## Stored Settings
By default, `replicatr` creates a profile folder in `$HOME/replicatr`. use the `-p` or `--profile` folder to change the
name; the location is not configurable, however. Add a dot prefix to have it hidden from regular directory listings eg
`-p .localcache`

Inside this folder will be stored two configuration files, `config.json` and `info.json`. The first contains persistent
settings based on the CLI arguments shown above, and the second contains a
[NIP-11](https://github.com/nostr-protocol/nips/blob/master/11.md) Relay Information Document. Any relevant parameters
set on the command line will override those found in these two files (such as `--auth`) for the duration of the run. See below for all configurable parameters and subcommands.


### Options

#### Network and Identity
- **`--listen, -l`**  
  Sets the network address that the relay listens on.
  - **Example**: `-l 0.0.0.0:8080`

- **`--canisteraddr, -C`**  
  Specifies the IC canister address to use. 
  - **Local**: `-C http://127.0.0.1:<Port Number>`
  - **Main Net**: `-C https://icp0.io/`

- **`--canisterid, -I`**  
  The IC canister ID to be used.
  - **Example**: `-I abcde-fghij-klmno-pqrst-uvw`

- **`--seckey, -s`**  
  Identity key of the relay, used for various security functions.
  - **Example**: `-s mySecretKey`

#### Configuration and Metadata
- **`--profile, -p`**  
  Defines the profile name to use for storage, defaults to `replicatr`.
  - **Example**: `-p myProfile`

- **`--name, -n`**  
  Sets the name of the relay for NIP-11.
  - **Example**: `-n MyRelay`

- **`--description, -d`**  
  Provides a description of the relay for NIP-11.
  - **Example**: `-d "My custom relay"`

- **`--pubkey`**  
  Public key of the relay operator.
  - **Example**: `--pubkey 123abc`

- **`--contact, -c`**  
  Contact details of the relay operator.
  - **Example**: `-c email@example.com`

- **`--icon, -i`**  
  Icon to show on relay information pages.
  - **Example**: `-i http://example.com/icon.png`

#### Access Control
- **`--auth, -a`**  
  Requires NIP-42 authentication for all access.
  - **Example**: `-a`

- **`--public`**  
  Allows public read access to users not on the access control list.
  - **Example**: `--public`

- **`--owner, -o`**  
  Specifies public keys of users with owner-level permissions on the relay.
  - **Example**: `-o pubkey1,pubkey2`

- **`--whitelist, -w`**  
  IP addresses that are only allowed to access.
  - **Example**: `-w 192.168.1.1`

- **`--allow, -A`**  
  IP addresses that are always allowed to access.
  - **Example**: `-A 192.168.1.1`

#### Performance and Resource Management
- **`--sizelimit, -S`**  
  Sets the maximum size of the event store in bytes.
  - **Example**: `-S 1000000`

- **`--lowwater, -L`**  
  Target percentage for database size during garbage collection.
  - **Example**: `-L 20`

- **`--highwater, -H`**  
  Trigger percentage for database size during garbage collection.
  - **Example**: `-H 80`

- **`--gcfreq, -G`**  
  Frequency in seconds to check if database needs garbage collection.
  - **Example**: `-G 3600`

- **`--maxprocs`**  
  Maximum number of goroutines to use.
  - **Example**: `--maxprocs 8`

- **`--loglevel`**  
  Sets the log level.
  - **Options**: `off`, `fatal`, `error`, `warn`, `info`, `debug`, `trace`
  - **Example**: `--loglevel debug`

- **`--pprof`**  
  Enables CPU and memory profiling.
  - **Example**: `--pprof`

- **`--gcratio`**  
  GC percentage for triggering GC sweeps.
  - **Example**: `--gcratio 50`

- **`--memlimit`**  
  Sets a memory limit on the process to constrain memory usage.
  - **Example**: `--memlimit 500MB`

- **`--pollfrequency`**  
  Frequency of polling if a level 2 event store is enabled.
  - **Example**: `--pollfrequency 5`

- **`--polloverlap`**  
  Overlap multiple of poll frequency to account for latency.
  - **Example**: `--polloverlap 2`

### Commands

- **`initcfg`**  
  Initialize relay configuration files.

- **`export`**  
  Export database as line-structured JSON.

- **`import`**  
  Import data from line-structured JSON.

- **`pubkey`**  
  Print relay canister public key.

- **`addrelay`**  
  Add a relay to the cluster.  
  - **`--addpubkey`**  
    Public key of the client to add.  
    - **Example**: `addrelay --addpubkey 987xyz`
  - **`--admin`**  
    Set client as an admin.  
    - **Example**: `add relay --addpubkey 987xyz --admin`

- **`removerelay`**  
  Remove a relay from the cluster.  
  - **`--removepubkey`**  
    Public key of the client to remove.  
    - **Example**: `removerelay --removepubkey 987xyz`

- **`getpermission`**  
  Get permission of a relay.

- **`wipebdb`**  
  Empties local badger database (bdb).

> Note: all subcommands will execute the command and exit. The relay will not continue to run as in the general case.

