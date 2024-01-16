# postr

nostr CLI client written in Go

forked from [algia](github.com/mattn/algia)

## Usage

```
NAME:
   postr - A cli application for nostr

USAGE:
   postr [global options] command [command options] 

DESCRIPTION:
   A cli application for nostr

COMMANDS:
   timeline, tl  show timeline
   post, n       post new note
   reply, r      reply to the note
   repost, b     repost the note
   like, l       like the note
   search, s     search notes
   profile       show profile
   version       show version
   help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   -a value        profile appName
   --relays value  relays
   -V              verbose (default: false)
   --help, -h      show help
```

## Installation

Or install with go install command.

```
go install github.com/Hubmakerlabs/replicatr/cmd/postr@latest
```

## Configuration

Minimal configuration. Need to be at ~/.config/algia/config.json

```json
{
  "relays": {
    "wss://relay-jp.nostr.wirednet.jp": {
      "read": true,
      "write": true,
      "search": false
    }
  },
  "privatekey": "nsecXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
}
```

If you want to zap via Nostr Wallet Connect, please add `nwc-pub` and `nwc-uri`
which are provided from <https://nwc.getalby.com/apps/new?c=Algia>

```json
{
  "relays": {
   ...
  },
  "privatekey": "nsecXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
  "nwc-uri": "nostr+walletconnect://xxxxx",
  "nwc-pub": "xxxxxxxxxxxxxxxxxxxxxxx"
}
```

## TODO

* [x] like
* [x] repost
* [x] zap
* [x] upload images

## FAQ

Do you use proxy? then set environment variable `HTTP_PROXY` like below.

    HTTP_PROXY=http://myproxy.example.com:8080

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a. mattn)
