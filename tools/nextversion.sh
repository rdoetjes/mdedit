#!/bin/sh
# Prints the next release version: the highest v0.N.0 tag with N one higher,
# or 0.1.0 if there are no tags yet. Needs the tags (git fetch --tags).
set -e
last=$(git tag --list 'v0.*.0' | sed 's/^v0\.\([0-9]*\)\.0$/\1/' | grep -E '^[0-9]+$' | sort -n | tail -1)
echo "0.$(( ${last:-0} + 1 )).0"
