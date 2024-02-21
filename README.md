# replicatr

nostr relay with modular storage and connectivity

## about

`replicatr` is a `nostr` relay written in pure Go, aimed at becoming a single,
modular, and extensible reference implementation of the `nostr` protocol as
described in the
nostr [NIP (nostr implementation possibilities) specification](https://github.com/nostr-protocol/nips).

It will use a [badger](https://github.com/dgraph-io/badger)
data store for local caching, and interface with
the [internet computer](https://internetcomputer.org/) for storage of all
event types except ephemeral and private events.

## Notes about the logger

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

```json
[
  '([/]([a-zA-Z@0-9-_.]+/)+([a-zA-Z@0-9-_.]+)):([0-9]+)$,goland $1 --line $4,false',
  '([^/]([a-zA-Z@0-9-_.]+/)+([a-zA-Z@0-9-_.]+)):([0-9]+)$,openhyperlink $1 $4,false'
]
```

Two additional small scripts need to be added to your path and marked 
executable in order to allow you to change the absolute path prefix in the 
second of these two entries:

`openhyperlink` should look like this:
```bash
#!/usr/bin/bash
goland $(cat ~/.currpath)/$1 --line $2

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