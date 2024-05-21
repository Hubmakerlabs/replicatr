#!/bin/bash

# Attempt to find the project root by looking for a specific marker file or directory
while [[ $PWD != '/' && ! -f 'marker.file' ]]; do cd ..; done

# Check if we found the marker file, if not, exit
if [ ! -f 'marker.file' ]; then
    echo "Please run this script from within the project directory."
    exit 1
fi

# Run dfx start in the background
dfx start --background

# Define the dfx project directory
DFX_DIR="./cmd/canister/"

# Ensure the dfx directory exists
mkdir -p "$DFX_DIR"

# Define the target directory for the Rust source file relative to the current directory
TARGET_DIR="./cmd/canister/src/replicatr/src"

# Ensure the target directory exists
mkdir -p "$TARGET_DIR"

cd "$DFX_DIR"

# Build the project
dfx build

# Create the canister
dfx canister create replicatr --network=ic

# Save the canister ID as a variable
ID=$(dfx canister id replicatr --network=ic)

# Check if the ID was successfully retrieved
if [ -z "$ID" ]; then
    echo "Failed to retrieve canister ID"
    exit 1
fi

# Return to root directory using marker.file
while [[ $PWD != '/' && ! -f 'marker.file' ]]; do cd ..; done

# Run initcfg to initialize relay with canister_id and to generate secret key
go run . initcfg -I $ID

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

# Deploy the canister
dfx deploy --network=ic

# Ensure the canister_ids.json file exists
CANISTER_IDS_FILE="canister_ids.json"
if [ ! -f "$CANISTER_IDS_FILE" ]; then
    echo "{}" > "$CANISTER_IDS_FILE"
fi

# Update canister_ids.json
jq ".replicatr.ic = \"$ID\"" "$CANISTER_IDS_FILE" > tmp.$$.json && mv tmp.$$.json "$CANISTER_IDS_FILE"

echo "Deployment complete. Canister IDs have been updated."
