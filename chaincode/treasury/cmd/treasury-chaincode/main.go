// The treasury chaincode is the production anchoring backend, deployed to the
// permissioned Hyperledger Fabric network. It records Merkle-root anchors of
// posted-entry batches; the ledger gateway invokes CommitAnchor and public
// verifiers call VerifyRoot. No operational data or PII goes on chain.
package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"

	"github.com/opentreasury/opentreasury/chaincode/treasury/internal/contract"
)

func main() {
	chaincode, err := contractapi.NewChaincode(&contract.AnchorContract{})
	if err != nil {
		log.Fatalf("creating treasury chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Fatalf("starting treasury chaincode: %v", err)
	}
}
