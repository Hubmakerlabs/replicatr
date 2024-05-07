#!/bin/bash

# Attempt to find the project root by looking for a specific marker file or directory
while [[ $PWD != '/' && ! -f 'marker.file' ]]; do cd ..; done

# Check if we found the marker file, if not, exit
if [ ! -f 'marker.file' ]; then
    echo "Please run this script from within the project directory."
    exit 1
fi

# Define the target directory for the Rust source file relative to the current directory
TARGET_DIR="./cmd/testnet/src/testnet_backend/src"

# Ensure the target directory exists
mkdir -p "$TARGET_DIR"

# Execute the Go command and capture the output
# eventually change to
#TODO 
#pubkey=$(go run . pubkey --loglevel off)
pubkey=$(go run . pubkey  -I bkyz2-fmaaa-aaaaa-qaaaq-cai -C http://127.0.0.1:46373/ -s 546c81c7390cf75ebf592b9627e95c4d21495766de56090b10b7a3f197c98d3b --loglevel off)

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