# Relay-To-Canister Access Control
Relay-access to canisters are managed via Secp256k1 public keys (derived from the relay's secret key) which are then used to sign function calls to the canister. A signed timestamp is additionally inlcuded in the request to ensure the request's validity and to prevent replay attacks. The canister manages these public keys via [self-authenticating Principals](https://wiki.internetcomputer.org/wiki/Principal#:~:text=A%20self%2Dauthenticating%20principal%20is,reference%20a%20subnet%20or%20user.).

## Permission Levels
### Owner
An `Owner` relay is authorized to perform all actions in a replicatr canister. This includes any action related to relay synchronization and relay authorization.
>There can be multiple owners

### User
An `User` relay is only authorized to perform actions related to relay synchronization.

### Unauthorized
An `Unauthorized` relay is only able to access their permission level to the canister (which would be `Unauthorized) and no other data.

## Canister-Access Related Commands
Execute these commands by running the following in the repo root directory:

```bash
go run . <flags> <args> <command> <command flags>
```
> Note: all commands will execute the command and exit. The relay will not continue to run after the command as in the general case. Only one command should be given per call\
> \
> Note: additional flags can be added to command calls for additional configuration. See [here](pkg/config/base/README.md)

### Commands
- **`pubkey`**  
  Print relay's canister public key.

- **`addrelay`**  
  Add a relay to the cluster - only an `Owner` relay is authorized to do this 
  - **`--addpubkey`**  
    Public key of the client to add.  
    - **Example**: `addrelay --addpubkey 987xyz`
  - **`--admin`**  
    Set client as an admin.  
    - **Example**: `add relay --addpubkey 987xyz --admin`

- **`removerelay`**  
  Remove a relay from the cluster - only an `Owner` relay is authorized to do this
  - **`--removepubkey`**  
    Public key of the client to remove.  
    - **Example**: `removerelay --removepubkey 987xyz`

- **`getpermission`**  
  Obtain the access-level of a relay.
  - **Options**: `Owner`, `User`, `Unauthorized`
