# publicatr

nostr CLI client written in Go

forked from github.com/mattn/algia

## Usage

```
NAME:
   publicatr - A cli application for nostr

USAGE:
   publicatr [global options] command [command options] [arguments...]

DESCRIPTION:
   A cli application for nostr

COMMANDS:
   timeline, tl  show timeline
   stream        show stream
   post, n       post new note
   reply, r      reply to the note
   repost, b     repost the note
   unrepost, B   unrepost the note
   like, l       like the note
   unlike, L     unlike the note
   delete, d     delete the note
   search, s     search notes
   dm-list       show DM list
   dm-timeline   show DM timeline
   dm-post       post new note
   profile       show profile
   powa          post ぽわ〜
   puru          post ぷる
   zap           zap note1
   version       show version
   help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   -a value        profile name
   --relays value  relays
   -V              verbose (default: false)
   --help, -h      show help
```

## Installation

Install with go install command.

```
go install github.com/Hubmakerlabs/replicatr/cmd/publicatr@latest
```

## Configuration

Minimal configuration. Need to be at ~/.config/publicatr/config.json

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

If you want to zap via Nostr Wallet Connect, please add `nwc-pub` and `nwc-uri` which are provided from <https://nwc.getalby.com/apps/new?c=Algia>

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
David Vennik (a.k.a. mleku)
