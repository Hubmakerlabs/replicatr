# Troubleshooting Owner Setup 

If you have any issues with the set-up script, you can perform the setup manually. Here are the steps to do so:

## 1. Initialize Relay

Using the id from your previously created canister, run initcfg (from the root directory):

```bash
go run . initcfg -e ic -I <canister-id>
```

> This initializes the relay with the canister_id  and generates a secret key


## 2. Set Relay as Canister Owner

Run the following command to obtain your canister-facing relay pubkey:

 ```bash
 go run . pubkey
 ```

Copy and paste the resulting key into [replicatr/cmd/canister/src/replicatr/src/owner.rs](/cmd/canister/src/replicatr/src/owner.rs)

```rust
pub static OWNER: &str = "<canister-facing relay pubkey>";
```
> This sets your relay as the primary owner of the canister
> To learn more about canister permissions, [click here](doc/canister.md).

## 3. Create canister-ids.json

Create a file named canister-ids.json in [replicatr/cmd/canister](/cmd/canister)

Using the previously created canister, write the following in the file:

```json
{
  "replicatr": {
    "ic": "<canister-id>"
  }
}
```

## 4. Deploy Canister

From the root directory, run:

```bash
dfx start --clean --background
cd cmd/canister
dfx deploy replicatr --mode=reinstall --network=ic
```

> reinstall is needed if you already tried to run the set-up script
> this is because the primary relay owner is set only once when the canister is initially installed 




