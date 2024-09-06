#!/bin/sh

canisters=(
    "replicatr"
)

echo -e "${GREEN}> $ENV: Generating required files..${NC}"
dfx generate --network ic

for t in ${canisters[@]}; do
    echo -e "${GREEN} $ENV > Generating candid for $t..${NC}"
    cargo test candid -p $t
done

rm -rf src/declarations
echo -e "${GREEN} $ENV > Stopping local replica..${NC}"
