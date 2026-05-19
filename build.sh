#!/usr/bin/env bash
set -e
OUT="dist"
mkdir -p "$OUT"
echo "Building linux/amd64..."
GOOS=linux GOARCH=amd64 go build -o "$OUT/splitbandwidth_linux_amd64" .
echo "Building darwin/arm64..."
GOOS=darwin GOARCH=arm64 go build -o "$OUT/splitbandwidth_darwin_arm64" .
echo "Done:"
ls -lh "$OUT"/
