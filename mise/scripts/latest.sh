#!/usr/bin/env bash

curl -s "https://api.github.com/repos/ChainSafe/vm-compat/releases/latest" | jq -r '.tag_name' | sed 's/^v//'