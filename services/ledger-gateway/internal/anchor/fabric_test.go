package anchor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeContract stands in for the Fabric gateway contract client.
type fakeContract struct {
	submitted [][]string
	err       error
	txID      string
}

func (f *fakeContract) Submit(name string, args ...string) ([]byte, string, error) {
	f.submitted = append(f.submitted, append([]string{name}, args...))
	if f.err != nil {
		return nil, "", f.err
	}
	return nil, f.txID, nil
}

func TestFabricBackendCommitsRootAsAnchorID(t *testing.T) {
	contract := &fakeContract{txID: "tx-123"}
	backend := NewFabricBackend(contract, "treasury", "opentreasury")

	ref, err := backend.Commit(context.Background(), "root-abc", 7)
	require.NoError(t, err)
	require.Equal(t, "fabric:opentreasury:tx-123", ref)

	require.Len(t, contract.submitted, 1)
	call := contract.submitted[0]
	require.Equal(t, "CommitAnchor", call[0])
	// The Merkle root is its own anchor id: deterministic, so a crash-retry
	// of the same batch converges instead of double-anchoring.
	require.Equal(t, "root-abc", call[1], "anchor id must be the root")
	require.Equal(t, "root-abc", call[2], "merkle root argument")
	require.Equal(t, "7", call[3], "entry count travels on chain")
}

func TestFabricBackendTreatsAlreadyExistsAsSuccess(t *testing.T) {
	contract := &fakeContract{err: errors.New(`chaincode response 500, anchor "root-abc" already exists`)}
	backend := NewFabricBackend(contract, "treasury", "opentreasury")

	ref, err := backend.Commit(context.Background(), "root-abc", 7)
	require.NoError(t, err, "recommitting an existing anchor must converge, not fail")
	require.Equal(t, "fabric:opentreasury:root-abc", ref)
}

func TestFabricBackendPropagatesOtherErrors(t *testing.T) {
	contract := &fakeContract{err: errors.New("endorsement failure")}
	backend := NewFabricBackend(contract, "treasury", "opentreasury")

	_, err := backend.Commit(context.Background(), "root-abc", 7)
	require.ErrorContains(t, err, "endorsement failure")
}

func TestFabricBackendName(t *testing.T) {
	require.Equal(t, "fabric", NewFabricBackend(&fakeContract{}, "treasury", "opentreasury").Name())
}
