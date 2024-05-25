# Blower: A Nostr Event Uploader

## Overview
Blower is a command-line tool that facilitates the uploading of Nostr events from a JSONL file to a specified Nostr relay. This tool is part of the `replicatr` package and is designed to work seamlessly with Nostr network protocols to manage events efficiently.

## Features
- **Uploads Events:** Pushes events from a local JSONL file to a Nostr relay.
- **Configuration Management:** Saves and retrieves configuration settings to simplify recurring operations.
- **Secure Connection:** Utilizes Nostr keys for authentication, ensuring secure interaction with the relay.


## Usage
### Basic Command Structure

```bash
go run ./blower --nsec [NSEC] --uploadrelay [RELAY_URL] --sourcefile [PATH_TO_JSONL_FILE]
```

### Parameters
- **`--nsec`** (required if not saved): Your Nostr secret key, encoded in Bech32 format. This is used for authentication with the relay.
- **`--uploadrelay`** (required): The URL of the Nostr relay where events will be uploaded.
- **`--sourcefile`** (required): The path to the `.jsonl` file containing Nostr events to push to the relay.
- **`--pause`** (optional): Time in milliseconds to wait between requests, to accommodate rate limits of the relay.
- **`--skip`** (optional): Number of events to skip from the beginning of the file, useful for resuming uploads.

### Configuration Files
Blower can save your Nostr secret key in a configuration file within your home directory under `~/.vacuum.json`. This allows you to run the tool without specifying the `--nsec` parameter each time.

### Running the Tool
To upload events to a relay, ensure your JSONL file is formatted correctly with valid Nostr events. Here is an example command:

```bash
go run ./blower --nsec your_bech32_encoded_key --uploadrelay https://relay.example.com --sourcefile path/to/events.jsonl
```
This command will start pushing events from events.jsonl to https://relay.example.com.

### Error Handling
Blower is equipped with comprehensive logging and error handling to ensure that any issues during the upload process are clearly communicated. It will attempt to reconnect in case of network issues or interruptions.
