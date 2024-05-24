# access control lists

> NOT YET FULLY IMPLEMENTED

Access Control Lists are a data structure for storing the user privileges for a
computer system. In `replicatr` this applies to several use cases, mainly the 
blacklisting use case, and the paid relay use case.

To implement this, in `replicatr` we use a special type of internal event 
that is stored directly by the relay in response to interactions that take place
in Direct Messages sent by users.

Rather than build a whole client infrastructure to implement this, it is simpler
to simply build a chat bot interface, with the relay having a specific user 
identity, and when a user DMs the relay identity, it detects it is being 
messaged and processes the input and returns a response to the message as 
well as whatever relevant processing that takes place.

A special event kind has been created for this purpose, with the kind number 
39998 - a number that hopefully will not be interesting to anyone until 
after this protocol becomes part of the specification, if it does ever, and 
if not, the relay can always have special handling to recognise these events 
by the combination of the kind number *and* the pubkey publishing them, 
which is the relay's chat identity public key.

The specification of the event data structure is as follows:

```json
{
  "id": "event ID",
  "pubkey": "relay pubkey",
  "created_at": 1708516744,
  "kind": 39998,
  "tags": [
    [
      "p",
      "pubkey of user",
      "role as string eg: 'reader'"
    ],
    [
      "replaces",
      "id of previous event that set a role for this pubkey"
    ],
    [
      "expiry",
      "unix timestamp as string for expiry, if none set, no change to previous"
    ]
  ],
  "content": "",
  "sig": "event signature"
}
```

As is the convention with most event types, the public key is tagged with a 
"p" tag, and then there is two more tags that can appear:

- "replaces" - if there is an existing previous record that has a "p" tag 
  the same as this current event, the Event ID of this tag comes after it.
- "expiry" - a timestamp as a decimal representation of the unix timestamp 
  representing the expiry time of the role, after which the role reverts to 
  "none" by default

The role strings are as follows:	
- "owner"
- "admin"
- "writer"
- "reader"
- "denied"
- "none"

At startup the relay will search for all stored events with this kind, 
published by the relay's pubkey, and use this to populate the in-memory form 
of the ACL that is then used by the filters to process or reject envelopes 
received by the relay.

When the user is not authenticated, they are treated as though they have 
"none" role, which in an auth-required relay means no access, when they are 
authenticated, their public key is searched for in the ACL and the role they 
have at the current time of this check is enforced.