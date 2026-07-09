// Package contract implements the OpenTreasury treasury chaincode: the
// production anchoring backend. It records Merkle roots of posted-entry
// batches on the permissioned Hyperledger Fabric ledger. No operational data,
// PII, or non-public field ever goes on chain — only hashes and public-safe
// anchor metadata, per the classification policy.
package contract

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// AnchorContract records and reads Merkle-root anchors.
type AnchorContract struct {
	contractapi.Contract
}

// Anchor is the on-chain commitment for one batch of posted journal entries.
type Anchor struct {
	ID          string `json:"id"`
	MerkleRoot  string `json:"merkleRoot"`
	EntryCount  int    `json:"entryCount"`
	Period      string `json:"period"`
	CommittedAt string `json:"committedAt"`
}

const anchorKeyPrefix = "anchor"

// CommitAnchor records a Merkle root. Anchors are immutable: committing the
// same id twice is rejected, so history cannot be rewritten.
func (c *AnchorContract) CommitAnchor(ctx contractapi.TransactionContextInterface, id, merkleRoot string, entryCount int, period string) error {
	if id == "" || merkleRoot == "" {
		return fmt.Errorf("anchor id and merkleRoot are required")
	}

	key, err := ctx.GetStub().CreateCompositeKey(anchorKeyPrefix, []string{id})
	if err != nil {
		return err
	}

	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("anchor %q already exists", id)
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}

	anchor := Anchor{
		ID:          id,
		MerkleRoot:  merkleRoot,
		EntryCount:  entryCount,
		Period:      period,
		CommittedAt: timestamp.AsTime().UTC().Format("2006-01-02T15:04:05Z"),
	}
	payload, err := json.Marshal(anchor)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(key, payload)
}

// GetAnchor returns a previously committed anchor, or an error if absent.
func (c *AnchorContract) GetAnchor(ctx contractapi.TransactionContextInterface, id string) (*Anchor, error) {
	key, err := ctx.GetStub().CreateCompositeKey(anchorKeyPrefix, []string{id})
	if err != nil {
		return nil, err
	}

	payload, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, fmt.Errorf("anchor %q does not exist", id)
	}

	var anchor Anchor
	if err := json.Unmarshal(payload, &anchor); err != nil {
		return nil, err
	}
	return &anchor, nil
}

// VerifyRoot reports whether the given root matches the anchor's committed
// root — the on-chain check a public verifier can call.
func (c *AnchorContract) VerifyRoot(ctx contractapi.TransactionContextInterface, id, merkleRoot string) (bool, error) {
	anchor, err := c.GetAnchor(ctx, id)
	if err != nil {
		return false, err
	}
	return anchor.MerkleRoot == merkleRoot, nil
}
