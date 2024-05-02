# replicatr

## [nostr](https://nostr.com/) [layer2](#what-is-layer2)

a nostr relay designed to enable modular data storage and connectivity for a borderless, composable global social network.

Nostr is the base layer, and what Replicatr brings is the **Layer 2** enabling the data to be separated from the delivery.

With Replicatr, Nostr can scale up to massive, heteregenous islands in a great archipelago of social networks, with low friction for users to transit or interact with multiple communities concurrently.

# sponsors

We gratefully acknowledge 

### ![](https://cdn-assets-eu.frontify.com/s3/frontify-enterprise-files-eu/eyJwYXRoIjoiZGZpbml0eVwvZmlsZVwvZmE0QTVhcUR4MWVWZVJFQTRiTnAucG5nIn0:dfinity:IdAJOMHSBmHNqnd87mG-FQjWJO9E7dGTG802kJeqRTk?width=32) Dfinity and the Internet Computer Protocol

who provided the funding and framework to execute this project

### ![](https://aqs24-xaaaa-aaaal-qbbea-cai.ic0.app/logos/catalyze-mini.svg) Hubmakerlabs and Catalyze One

who organised this project and brought together the developers and administrators required to make it happen

# about

`replicatr` is a `nostr` relay written in pure Go, aimed at becoming a single,
modular, and extensible reference implementation of the `nostr` protocol as
described in the
nostr [NIP (nostr implementation possibilities) specification](https://github.com/nostr-protocol/nips).

It will use a [badger](https://github.com/dgraph-io/badger)
key/value store for local caching, and interface, and be designed to integrate with **layer 2** data storage systems such as [the internet computer protocol](https://internetcomputer.org) - and to bring the borderless, low friction connectivity of Nostr to the world.

# what is **layer2**?

## `nostr` relays as distributed caches for (multiple) larger decentralized data stores

The reason why the name of this relay project is `replicatr` is in reference to the distributed database technology term *replica* which is the name given to a node in a distributed database system that contains a copy, or "replica" of the same data as the other nodes in the database cluster.

Part of the details of implementing a two level storage system is the idea of creating bounds on the expansion of storage utilization in one layer and scalability questions on the second level.

Mutable large data stores like the one being implemented for the initial beta of `replicatr`, the Internet Computer Protocol are one strategy for the *layer2* of `nostr`, but immutable stores with dynamic replication strategies that scale up replicas of data that are in demand and discard data that is of lesser importance, such as IPFS and Arweave and other content addressable storage strategies.

It is entirely conceivable, even, to even go to *layer3* and *layer4* with event storage, but really, it makes no sense to add this distinction, when *layer2* can implicitly mean anything beyond the hot cache of a relay.

## developer notes

### notes about the logger

Due to its high performance at rendering and its programmable custom 
hyperlink capability, VTE based terminal 
[Tilix](https://github.com/gnunn1/tilix) is the best option for Linux 
developers, if you are on windows or mac, there is options but the main 
author of this repo doesn't and refuses to use such abominations.

The performance of the Goland terminal, which does this by default and 
manages its relative path interpretation based on the current opened project,
is abysmal if there is long lines, probably due to it being written in 
highly abstracted Java rather than C like VTE's rexept hyperlink engine.

So if you use VSCode or other non-Goland IDE, you may want to change the 
invocation in the following command and script to fit the relevant too; the 
provided versions here work with Goland so long as it has had a `goland` 
launch script deployed to your `$PATH` somewhere.

The following are a pair of custom hyperlink specifications, extracted using 
dconf-editor, from the path /com/gexperts/Tilix/custom-hyperlinks that works 
to give you absolute and relative paths when you are using the 
[slog](https://mleku.dev/git/slog) logger that is used throughout this 
project and also on several of the dependencies that live at the same git 
hosting address:

```
[
  '([/]([a-zA-Z@0-9-_.]+/)+([a-zA-Z@0-9-_.]+)):([0-9]+)$,goland --line $4 $1,false',
  '([^/]([a-zA-Z@0-9-_.]+/)+([a-zA-Z@0-9-_.]+)):([0-9]+),openhyperlink $1 $4,false'
]
```

Two additional small scripts need to be added to your path and marked 
executable in order to allow you to change the absolute path prefix in the 
second of these two entries:

`openhyperlink` should look like this:
```bash
#!/usr/bin/bash
goland --line $2 $(cat ~/.currpath)/$1

```
and to set that `.currpath` file to contain a useful path:

`currpath`

```bash
#!/usr/bin/bash
echo $(pwd)>~/.currpath
echo .currpath set to $(< ~/.currpath)

```

This will then assume any relative code locations like `app/broadcasting.go:32`
will have the value from `~/.currpath` prefixed in, and you can invoke 
`currpath` at the root of your repository in order to have the relative 
paths work.

The logger doesn't generate relative paths, as this is an additional 
complexity between the logger code and the environment that is not worth 
doing anything about, you could add an invocation of `currpath` to your `.
bashrc` and when you open the terminal in that location it would 
automatically be set, but if you open a terminal elsewhere it would 
overwrite it.

The reason for having the relative paths is that when you execute your code 
if there is syntax or other errors that prevent compilation, the Go tooling 
prints them as module-relative paths, which also may get confusing if you 
have got a project with multiple go modules in it.

I personally believe that there should be one `go.mod` in a project, as I 
have seen the results of this in the `btcd` and `lnd` projects and it has 
led to multiple cases of self-imports of different versions from the same 
codebase, which is an abomination and the go modules equivalent of 
spaghetti - how are you going to debug that mess?
