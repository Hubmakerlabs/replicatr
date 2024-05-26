#!/bin/bash

# Attempt to find the project root by looking for a specific marker file or directory
while [[ $PWD != '/' && ! -f 'marker.file' ]]; do cd ..; done

# Check if we found the marker file, if not, exit
if [ ! -f 'marker.file' ]; then
    echo "Please run this script from within the project directory."
    exit 1
fi

# Run dfx start in the background
dfx start --background --clean

# Define the dfx project directory
DFX_DIR="./cmd/canister/"

# Ensure the dfx directory exists
mkdir -p "$DFX_DIR"

# Define the target directory for the Rust source file relative to the current directory
TARGET_DIR="./cmd/canister/src/replicatr/src"

# Ensure the target directory exists
mkdir -p "$TARGET_DIR"



# Prompt the user to input the canister ID
read -p "Please enter the canister ID: " ID

# Check if the ID was successfully provided
if [ -z "$ID" ]; then
    echo "Canister ID cannot be empty"
    exit 1
fi
# Run initcfg to initialize relay with canister_id and to generate secret key
go run . initcfg -I $ID -e ic

# Execute the Go command and capture the output (pubkey derived from secret key)
pubkey=$(go run . pubkey --loglevel off)

# Check if the pubkey was successfully retrieved
if [ -z "$pubkey" ]; then
    echo "Failed to retrieve public key"
    exit 1
fi

# Navigate to the target directory
cd "$TARGET_DIR"

# Create the owner.rs file and write the pubkey into it
echo "pub static OWNER: &str = \"$pubkey\";" > owner.rs

echo "owner.rs created successfully with public key at $TARGET_DIR/owner.rs."

# Cd out from "./cmd/canister/src/replicatr/src" to "./cmd/canister"
cd ../../..



# Ensure the canister_ids.json file exists
CANISTER_IDS_FILE="canister_ids.json"
if [ ! -f "$CANISTER_IDS_FILE" ]; then
    echo "{}" > "$CANISTER_IDS_FILE"
fi

# Update canister_ids.json
jq ".replicatr.ic = \"$ID\"" "$CANISTER_IDS_FILE" > tmp.$$.json && mv tmp.$$.json "$CANISTER_IDS_FILE"

# Deploy the canister
dfx deploy replicatr --network=ic

echo "Relay initialized. Deployment complete. Canister ID updated."
