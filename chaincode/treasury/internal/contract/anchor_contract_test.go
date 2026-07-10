package contract

import (
	"testing"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// minimal in-memory stub implementing the ChaincodeStubInterface subset the
// contract uses.
type memStub struct {
	shim.ChaincodeStubInterface
	state map[string][]byte
}

func newStub() *memStub { return &memStub{state: map[string][]byte{}} }

func (s *memStub) CreateCompositeKey(prefix string, attrs []string) (string, error) {
	return prefix + "\x00" + attrs[0], nil
}
func (s *memStub) GetState(key string) ([]byte, error) { return s.state[key], nil }
func (s *memStub) PutState(key string, value []byte) error {
	s.state[key] = value
	return nil
}
func (s *memStub) GetTxTimestamp() (*timestamppb.Timestamp, error) {
	return timestamppb.New(time.Unix(1_700_000_000, 0)), nil
}

type memCtx struct {
	contractapi.TransactionContextInterface
	stub *memStub
}

func (c *memCtx) GetStub() shim.ChaincodeStubInterface { return c.stub }

var _ = queryresult.KV{}
var _ = peer.Response{}

func newCtx() *memCtx { return &memCtx{stub: newStub()} }

func TestCommitAndGetAnchor(t *testing.T) {
	contract := &AnchorContract{}
	ctx := newCtx()

	require.NoError(t, contract.CommitAnchor(ctx, "anc-1", "root-abc", 3, "2026-07"))

	anchor, err := contract.GetAnchor(ctx, "anc-1")
	require.NoError(t, err)
	require.Equal(t, "root-abc", anchor.MerkleRoot)
	require.Equal(t, 3, anchor.EntryCount)
	require.Equal(t, "2026-07", anchor.Period)
}

func TestAnchorsAreImmutable(t *testing.T) {
	contract := &AnchorContract{}
	ctx := newCtx()

	require.NoError(t, contract.CommitAnchor(ctx, "anc-1", "root-abc", 3, "2026-07"))
	err := contract.CommitAnchor(ctx, "anc-1", "root-different", 5, "2026-07")
	require.Error(t, err, "re-committing an anchor id must be rejected")
}

func TestVerifyRoot(t *testing.T) {
	contract := &AnchorContract{}
	ctx := newCtx()
	require.NoError(t, contract.CommitAnchor(ctx, "anc-1", "root-abc", 3, "2026-07"))

	ok, err := contract.VerifyRoot(ctx, "anc-1", "root-abc")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = contract.VerifyRoot(ctx, "anc-1", "root-tampered")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestGetMissingAnchorErrors(t *testing.T) {
	contract := &AnchorContract{}
	ctx := newCtx()

	_, err := contract.GetAnchor(ctx, "nope")
	require.Error(t, err)
}
