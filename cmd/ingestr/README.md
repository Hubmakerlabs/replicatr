# ingestr

a tool to pull events related to your public key from one nostr relay and publish them to another

supports NIP-42 authentication

```
Usage: ingestr [--nsec NSEC] [--since SINCE] [--kinds KINDS] [--limit LIMIT] [--getfollows] DOWNLOADRELAY UPLOADRELAY

Positional arguments:
  DOWNLOADRELAY
  UPLOADRELAY

Options:
  --nsec NSEC, -n NSEC   use the nsec (bech32 encoded) for auth and if given, writes it to configuration and will be loaded afterwards until a new one is given
  --since SINCE, -s SINCE
                         only query events since this unix timestamp
  --kinds KINDS, -k KINDS
                         comma separated list of kind numbers to ingest
  --limit LIMIT, -l LIMIT
                         maximum of number of events to return for each day long interval [default: 1000]
  --getfollows, -g       also get follows' events'
  --help, -h             display this help and exit
```

requests the events in a sliding window of time from SINCE of 1 hour so that 
it caches everything

can be set to request any number of event kinds