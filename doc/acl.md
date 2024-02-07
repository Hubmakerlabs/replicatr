# access control

A minimal ACL file looks like this, stored in the location `~/replicatr` by
default or if on windows or mac can be other places - set environment
GODEBUG="debug" to see it print the path at startup.

```json
{
  "users": [
    {
      "role": "owner",
      "pubkey": "4c800257a588a82849d049817c2bdaad984b25a45ad9f6dad66e47d3b47e3b2f"
    }
  ],
  "public": true,
  "public_auth_required": false
}
```

This file can be generated automatically for you with the subcommond
`initacl` which takes one positional parameter, the hex encoded pubkey of
the primary owner, and an optional flag `--public` which creates an empty
reader key as above.

There is 4 types of privilege in the file, these are:

- `owner` - permitted to add, change and remove all other entries except
  owner itself (owner should have the ability to manually edit the
  configuration to change owners)
- `administrator` - permitted to read and publish events, add and change
  readers and writers
- `writer` - permitted to read and publish events to the relay
- `reader` - permitted to read events from the relay

If it is desired that the relay respond to queries from the public, set the `public` field to true, if you want to enforce authentication for public access, set `public_auth_required` to true. Both fields are optional to omit.

All access to privileged events, such as application specific data and
private messages require authentication and the authenticated pubkey must
match a party in the filters and events that will be returned.

When public auth is required, the possibility of rate limiting per user more 
exactly becomes possible, otherwise the rate limiter would rely on IP 
addresses as identifiers.

TODO: there is no rate limiting presently, 
implement this with a per-ip and per-pubkey basis.


