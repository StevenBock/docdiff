#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

mkdir -p "$INSTALL_DIR"
go build -o "$INSTALL_DIR/docdiff" ./cmd/docdiff
chmod +x "$INSTALL_DIR/docdiff"

echo "Installed docdiff to $INSTALL_DIR/docdiff"
