# `testr`

The `testr` package provides a powerful framework for building and executing highly customizable and exhaustive test cases for relay systems. It focuses on minimal disk usage and high configurability, allowing extensive automated testing of relay features.

## Features

- **Minimal Disk Usage**: Test cases are built and executed on the fly, avoiding the need to save tests to disk. This feature supports the creation of large numbers of test cases, which can be either seeded or randomly generated.

- **Dual Mode Operation**: 
  - **Test Instance Mode** (default): Creates an alternate instance of the relay and database with a test canister. This mode includes a complete setup and cleanup process, ensuring a clean testing environment.
  - > When running in test instance mode, the canister-id will have to be provided twice: first to the `testr` so it can perform its clean up, and then to the relay as usual. See below for details.
  - > A second test canister will have to be created via [NNS](https://nns.ic0.app/) to most effectively use this mode
  - **Primary Instance Mode**: Uses the primary relay and canister for testing, directly connecting to the relay via WebSockets to validate behavior.
    - To test on primary instance mode, simply change your profile (using -p) to your main profile (`replicatr` by default) when prompted to enter your relay run command
    - > When testing on your primary instance ensure that --skipsetup is used to ensure your database and canister are not wiped before and after the test.

- **Isolated Database Instance**: Utilizes a separate BadgerDB instance to mirror data handling in the relay, allowing for parallel querying and result comparison between the relay and the database.

## Usage

Run the package from the root directory using the following format:

```bash
go run ./cmd/testr <flags> <args>
```

### Configurable Options

- `--seed SEED, -s SEED`: Seed for random generation (integer).
- `--events EVENTS, -e EVENTS`: Number of events to generate [default: 50].
- `--queries QUERIES, -q QUERIES`: Number of queries to generate [default: 50].
- `--skipsetup`: Skip the pre and post test setup/cleanup [default: false].
- `--canisteraddr CANISTERADDR, -C CANISTERADDR`: Address of the IC canister [default: https://icp0.io/].
- `--canisterid CANISTERID, -I CANISTERID`: ID of the IC canister [default: rpfa6-ryaaa-aaaap-qccvq-cai].
- `--wipe`: Enable wiping of the canister and BadgerDB [default: false].
- `--loglevel LOGLEVEL`: Logging level (off, fatal, error, warn, info, debug, trace) [default: info].
- `--seckey SECKEY, -s SECKEY`: Security key for relay control.
> Note: When running in the test-environment mode, settings such as the secret key, canister ID, etc., need to be provided as flags as the setup/cleanup ensure no settings are saved from previous calls.


### Relay Configurations
In addition to the `testr` configurations, all the relay configurations mentioned [here](pkg/config/base/README.md) apply as well.\
Upon running the `testr` package, you will be prompted with:

```bash
Enter command to run relay as usual with flags and args as needed:
```

where you should input your general relay run command formatted as:

```bash
go run . <flags> <args>
```

# Example

Example command to run `testr` in test instance mode (from the replicatr root directory):
```bash
go run ./cmd/testr --seed 12345 --events 100 --queries 100 -I fdasf-fdeyd-eregv-vxzeh
Enter command to run relay as usual with flags and args as needed:
go run . -I fdasf-fdeyd-eregv-vxzeh
```




