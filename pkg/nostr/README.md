
![nostr canary](./icon.png)

# nostr

Tools for structuring, processing and encoding things for the nostr protocol.

This covers almost everything found in
[go-nostr](https://github.com/nbd-wtf/go-nostr) except rewritten in fully
idiomatic Go and
restructured for better readability and organisation for a more friendly
developer experience.

[nostr-sdk](https://github.com/nbd-wtf/nostr-sdk) is also included as it 
contains useful tools for client developers and is required for the 
[algia](https://github.com/mattn/algia) fork `postr` found in 
[replicatr](https://github.com/Hubmakerlabs/replicatr)

The [eventstore](https://mleku.dev/git/nostr/eventstore) is also found in 
here, but with only the badger back end and the hand coded, badly written 
binary encoding format for events is replaced with the use of the Gob codec, 
which is fine for internal data stores used only by a Go based project. 

Perhaps later a full implementation of a gRPC/protobuf form of the entire 
protocol will follow, as it has performance advantages over JSON and broadly 
supported even by javascript based applications.

There is not very much documentation apart from a moderate effort to put 
proper godoc comments on everything, better documentation will come later.

Currently there is a set of packages named after NIPs, these will be 
refactored to have human readable names in the future, and for the support 
of any downstream users of this library, the existing references will be 
maintained via the use of symlinks and aliases where applicable with 
deprecation notices added when this is done.

This repo is being released independently as a service to fellow nostr 
developers who also like to use Go, as it is becoming stable and has 
numerous bugs from go-nostr fixed and has been restructured to be easier to 
navigate.

## Usage

Be aware that you should pick a specific semver version when importing, as 
the main branch is unstable. See [here](https://mleku.dev/git/nostr/tags) 
for current tags.

Currently this library is still a little unstable in general, probably not 
really strictly correct that it already has a v1 prefix, but it is a fork.

Technically it can be said that a v1.1.x version string implies instability, 
when the API is 100% stable (it mostly already is) we will bump it to v1.2.0.