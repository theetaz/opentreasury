package merkle

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleEntry() Entry {
	return Entry{
		ID: "je-1", InstitutionID: "minfin", FiscalYear: 2026, EffectiveDate: "2026-07-01",
		Status: "POSTED", EntryType: "STANDARD",
		Lines: []Line{
			{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 125000, Currency: "USD"},
			{AccountCode: "114", Direction: "CREDIT", AmountMinor: 125000, Currency: "USD"},
		},
	}
}

func TestCanonicalHashIsStableAndOrderIndependent(t *testing.T) {
	entry := sampleEntry()
	swapped := sampleEntry()
	swapped.Lines[0], swapped.Lines[1] = swapped.Lines[1], swapped.Lines[0]

	require.Equal(t, CanonicalHash(entry), CanonicalHash(swapped), "line order must not change the hash")
	require.Len(t, CanonicalHash(entry), 64)
}

func TestCanonicalHashChangesWhenAnyFieldChanges(t *testing.T) {
	base := CanonicalHash(sampleEntry())

	tampered := sampleEntry()
	tampered.Lines[0].AmountMinor = 125001
	require.NotEqual(t, base, CanonicalHash(tampered), "an amount change must change the hash")

	reclassified := sampleEntry()
	reclassified.Status = "REVERSED"
	require.NotEqual(t, base, CanonicalHash(reclassified))
}

func TestTreeProofVerifiesForEveryLeaf(t *testing.T) {
	// Use an odd count to exercise node promotion.
	leaves := []string{}
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		leaves = append(leaves, CanonicalHash(Entry{ID: id, Status: "POSTED", EntryType: "STANDARD"}))
	}

	tree := NewTree(leaves)
	root := tree.Root()
	require.NotEmpty(t, root)

	for i, leaf := range leaves {
		proof, err := tree.Proof(i)
		require.NoError(t, err)
		require.True(t, Verify(leaf, proof, root), "proof for leaf %d must verify", i)
	}
}

func TestVerifyRejectsTamperedLeaf(t *testing.T) {
	leaves := []string{
		CanonicalHash(Entry{ID: "a", Status: "POSTED", EntryType: "STANDARD"}),
		CanonicalHash(Entry{ID: "b", Status: "POSTED", EntryType: "STANDARD"}),
		CanonicalHash(Entry{ID: "c", Status: "POSTED", EntryType: "STANDARD"}),
	}
	tree := NewTree(leaves)
	root := tree.Root()

	proof, err := tree.Proof(0)
	require.NoError(t, err)

	tampered := CanonicalHash(Entry{ID: "a", Status: "REVERSED", EntryType: "STANDARD"})
	require.False(t, Verify(tampered, proof, root), "a tampered entry must fail verification")
}

func TestVerifyRejectsWrongRoot(t *testing.T) {
	leaves := []string{
		CanonicalHash(Entry{ID: "a", Status: "POSTED", EntryType: "STANDARD"}),
		CanonicalHash(Entry{ID: "b", Status: "POSTED", EntryType: "STANDARD"}),
	}
	tree := NewTree(leaves)
	proof, err := tree.Proof(0)
	require.NoError(t, err)

	require.False(t, Verify(leaves[0], proof, "0000000000000000000000000000000000000000000000000000000000000000"))
}

func TestSingleLeafTree(t *testing.T) {
	leaf := CanonicalHash(sampleEntry())
	tree := NewTree([]string{leaf})

	require.Equal(t, leaf, tree.Root())
	proof, err := tree.Proof(0)
	require.NoError(t, err)
	require.True(t, Verify(leaf, proof, tree.Root()))
}
