# nostr ACL

DRAFT

## Rationale

In order to simplify and enable the automation of access control to a relay, for purposes like paid relays moderated relays it is possible to define a set of events that alter the access control list rather than require direct intervention in the form of manually changing the configuration and restarting a relay.

## Access Levels

There is 4 basic levels of access:

- **owner** - who has full control, can see all events, and can alter the lower levels of permission arbitrarily - they cannot add or remove other owners, owners must be configured in a configuration file
- **administrator** - administrators can add and modify users access permissions or disable their access
- **writer** - has the right to publish events to the relay
- **reader** - has the right to read events from the relay
- **denied** - blacklisted keys that are not able to perform any action and any attempt to auth with them will lead to disconnection

When it is intended that read access to the relay not require explicit permission, it is still possible to require authentication and anyone can be a reader so any reader access permissions are meaningless in this case, but write access can be limited.

