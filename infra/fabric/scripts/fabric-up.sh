#!/usr/bin/env bash
# Brings up the local Fabric network and deploys the treasury chaincode
# (chaincode-as-a-service). Idempotent: re-running converges. Requires the
# main stack's Docker network (make up) to exist.
set -euo pipefail

FABRIC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE=(docker compose -f "$FABRIC_DIR/docker-compose-fabric.yaml")
TOOLS_IMAGE=hyperledger/fabric-tools:2.5
CHANNEL=opentreasury
CC_NAME=treasury
CC_VERSION="1.0"
CC_SEQUENCE=1
# The orderer listener runs TLS (required for Raft nodes); this CA validates it.
ORDERER_CA=/work/generated/crypto/ordererOrganizations/opentreasury.local/orderers/orderer.opentreasury.local/tls/ca.crt

if ! docker network inspect opentreasury_default >/dev/null 2>&1; then
  echo "the opentreasury Docker network does not exist — run 'make up' first" >&2
  exit 1
fi

echo "==> generating crypto material and channel genesis block"
if [ ! -d "$FABRIC_DIR/generated/crypto" ]; then
  docker run --rm -v "$FABRIC_DIR":/work -w /work "$TOOLS_IMAGE" \
    cryptogen generate --config crypto-config.yaml --output generated/crypto
fi
if [ ! -f "$FABRIC_DIR/generated/genesis.block" ]; then
  docker run --rm -v "$FABRIC_DIR":/work -w /work "$TOOLS_IMAGE" \
    configtxgen -configPath /work -profile OpenTreasuryGenesis \
    -channelID "$CHANNEL" -outputBlock generated/genesis.block
fi

echo "==> starting orderer, peer, and cli"
"${COMPOSE[@]}" up -d --wait orderer peer cli

echo "==> joining orderer and peer to channel $CHANNEL"
docker exec fabric-cli osnadmin channel join \
  --channelID "$CHANNEL" --config-block /work/generated/genesis.block \
  -o orderer.opentreasury.local:7053 \
  || echo "    (orderer already joined)"
docker exec fabric-cli peer channel join -b /work/generated/genesis.block \
  || echo "    (peer already joined)"

echo "==> packaging chaincode (ccaas)"
docker exec fabric-cli bash -c '
  set -euo pipefail
  mkdir -p /work/generated/pkg && cd /work/generated/pkg
  printf %s "{\"address\":\"treasury-chaincode:9999\",\"dial_timeout\":\"10s\",\"tls_required\":false}" > connection.json
  printf %s "{\"type\":\"ccaas\",\"label\":\"treasury_1.0\"}" > metadata.json
  tar -czf code.tar.gz connection.json
  tar -czf /work/generated/treasury.tar.gz metadata.json code.tar.gz
'

echo "==> installing chaincode on the peer"
docker exec fabric-cli peer lifecycle chaincode install /work/generated/treasury.tar.gz \
  || echo "    (already installed)"
PACKAGE_ID=$(docker exec fabric-cli peer lifecycle chaincode calculatepackageid /work/generated/treasury.tar.gz)
echo "    package id: $PACKAGE_ID"

echo "==> starting chaincode service"
CHAINCODE_ID="$PACKAGE_ID" "${COMPOSE[@]}" up -d --build treasury-chaincode

echo "==> approving and committing chaincode definition"
docker exec fabric-cli peer lifecycle chaincode approveformyorg \
  -o orderer.opentreasury.local:7050 --tls --cafile "$ORDERER_CA" \
  --channelID "$CHANNEL" --name "$CC_NAME" \
  --version "$CC_VERSION" --package-id "$PACKAGE_ID" --sequence "$CC_SEQUENCE" \
  || echo "    (already approved)"
docker exec fabric-cli peer lifecycle chaincode commit \
  -o orderer.opentreasury.local:7050 --tls --cafile "$ORDERER_CA" \
  --channelID "$CHANNEL" --name "$CC_NAME" \
  --version "$CC_VERSION" --sequence "$CC_SEQUENCE" \
  || echo "    (already committed)"

echo "==> smoke check: chaincode reachable through the peer"
docker exec fabric-cli peer chaincode query -C "$CHANNEL" -n "$CC_NAME" \
  -c '{"function":"GetAnchor","Args":["fabric-up-smoke"]}' 2>&1 \
  | grep -q "does not exist" \
  && echo "    chaincode responding (no smoke anchor, as expected)"

cat <<EOF

Fabric network is up. Point the ledger gateway at it with:
  docker compose -f infra/docker/docker-compose.yaml -f infra/docker/docker-compose.fabric.yaml up -d ledger-gateway
(or 'make fabric-gateway')
EOF
