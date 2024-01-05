#!/usr/bin/bash

for (( ; ; ));
do
  reset
  echo "hit Ctrl-C to recompile and launch replicatr; mash Ctrl-C to stop this script"
  go run ./cmd/replicatrd/.
done