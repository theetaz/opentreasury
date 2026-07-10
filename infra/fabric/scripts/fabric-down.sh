#!/usr/bin/env bash
# Tears down the local Fabric network. Pass --purge to also delete ledger
# volumes and generated crypto material.
set -euo pipefail

FABRIC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [ "${1:-}" = "--purge" ]; then
  docker compose -f "$FABRIC_DIR/docker-compose-fabric.yaml" down -v
  rm -rf "$FABRIC_DIR/generated"
else
  docker compose -f "$FABRIC_DIR/docker-compose-fabric.yaml" down
fi
