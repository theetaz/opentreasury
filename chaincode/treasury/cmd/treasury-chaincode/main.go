// The treasury chaincode is the production anchoring backend, deployed to the
// permissioned Hyperledger Fabric network. It records Merkle-root anchors of
// posted-entry batches; the ledger gateway invokes CommitAnchor and public
// verifiers call VerifyRoot. No operational data or PII goes on chain.
package main

import (
	"log"
	"os"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"

	"github.com/opentreasury/opentreasury/chaincode/treasury/internal/contract"
)

func main() {
	chaincode, err := contractapi.NewChaincode(&contract.AnchorContract{})
	if err != nil {
		log.Fatalf("creating treasury chaincode: %v", err)
	}

	// Chaincode-as-a-service: when the peer is configured with a ccaas
	// package, the chaincode runs as its own server and the peer connects to
	// it. Classic peer-launched mode remains the fallback.
	if address := os.Getenv("CHAINCODE_SERVER_ADDRESS"); address != "" {
		server := &shim.ChaincodeServer{
			CCID:     os.Getenv("CHAINCODE_ID"),
			Address:  address,
			CC:       chaincode,
			TLSProps: shim.TLSProperties{Disabled: true},
		}
		log.Print("treasury chaincode serving as ccaas")
		if err := server.Start(); err != nil {
			log.Fatalf("starting treasury chaincode server: %v", err)
		}
		return
	}

	if err := chaincode.Start(); err != nil {
		log.Fatalf("starting treasury chaincode: %v", err)
	}
}
