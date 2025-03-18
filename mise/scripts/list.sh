#!/usr/bin/env bash
curl -s "https://api.github.com/repos/ChainSafe/vm-compat/releases" | jq -r '.[].tag_name'