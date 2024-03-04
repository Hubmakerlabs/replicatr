#!/usr/bin/env bash
export NVM_DIR=${HOME}/.nvm
export NVM_VERSION=v0.39.1
export NODE_VERSION=18.1.0
# need to change ownership of /opt to the user you normally work with for this
export RUSTUP_HOME=/opt/rustup
export CARGO_HOME=/opt/cargo
export RUST_VERSION=1.76.0
export DFX_URL=https://github.com/dfinity/sdk/releases/download/0.17.0/dfx-0.17.0-x86_64-linux.tar.gz
export DFX=dfx-0.17.0-x86_64-linux.tar.gz
# get prerequisites (ubuntu 22.04)
sudo apt -yq update
sudo apt -yqq install --no-install-recommends curl ca-certificates build-essential \
    pkg-config libssl-dev llvm-dev liblmdb-dev clang cmake rsync wget
# install nodejs using nvm
export PATH="${HOME}/.nvm/versions/node/v${NODE_VERSION}/bin:${PATH}"
curl --fail -sSf https://raw.githubusercontent.com/creationix/nvm/${NVM_VERSION}/install.sh | bash
. "${NVM_DIR}/nvm.sh" && nvm install ${NODE_VERSION}
. "${NVM_DIR}/nvm.sh" && nvm use v${NODE_VERSION}
. "${NVM_DIR}/nvm.sh" && nvm alias default v${NODE_VERSION}
export NVM_DIR="$HOME/.nvm"
# This loads nvm
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
# This loads nvm bash_completion
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"
# install rust and IC dependencies
curl --fail https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain \
     ${RUST_VERSION}-x86_64-unknown-linux-gnu --no-modify-path
rustup default ${RUST_VERSION}-x86_64-unknown-linux-gnu
rustup target add wasm32-unknown-unknown
cargo install ic-wasm
# install dfx
DFX_VERSION=0.17.0 sh -ci "$(curl -fsSL https://raw.githubusercontent.com/dfinity/sdk/dfxvm-install-script/install.sh)"
source "$HOME/.local/share/dfx/env"

#dfx start --host 127.0.0.1:46847 --background
#dfx new testnet
#dfx deploy