package anchor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// fabricSubmitter is the narrow slice of the Fabric gateway contract client
// the backend needs: submit a transaction and learn its transaction id.
type fabricSubmitter interface {
	Submit(name string, args ...string) (result []byte, txID string, err error)
}

// FabricBackend commits Merkle roots to the treasury chaincode on a
// Hyperledger Fabric network (ADR-0004). The Merkle root doubles as the
// anchor id: it is deterministic for a batch, so a crash between the chain
// commit and the proof write converges on retry instead of double-anchoring.
type FabricBackend struct {
	contract  fabricSubmitter
	chaincode string
	channel   string
}

func NewFabricBackend(contract fabricSubmitter, chaincode, channel string) *FabricBackend {
	return &FabricBackend{contract: contract, chaincode: chaincode, channel: channel}
}

func (*FabricBackend) Name() string { return "fabric" }

func (b *FabricBackend) Commit(_ context.Context, merkleRoot string, entryCount int) (string, error) {
	_, txID, err := b.contract.Submit("CommitAnchor", merkleRoot, merkleRoot, strconv.Itoa(entryCount), "")
	if err != nil {
		// An anchor keyed by this root already exists on chain: a previous
		// run committed it but crashed before recording proofs. That is the
		// state we wanted — converge.
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Sprintf("fabric:%s:%s", b.channel, merkleRoot), nil
		}
		return "", fmt.Errorf("submitting CommitAnchor: %w", err)
	}
	return fmt.Sprintf("fabric:%s:%s", b.channel, txID), nil
}
