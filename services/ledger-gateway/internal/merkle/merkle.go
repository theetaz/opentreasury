// Package merkle implements the canonical entry hash, Merkle tree, inclusion
// proofs, and independent verification used by the traceability layer. Every
// function here is pure and deterministic so a third party can reproduce the
// exact same hashes and verify a proof without trusting OpenTreasury.
package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Entry is the canonical, public-safe view of a journal entry that gets
// anchored. It contains no free text — only structural, verifiable fields.
type Entry struct {
	ID            string
	InstitutionID string
	FiscalYear    int
	EffectiveDate string
	Status        string
	EntryType     string
	Lines         []Line
}

type Line struct {
	AccountCode string
	Direction   string
	AmountMinor int64
	Currency    string
}

// CanonicalHash produces the leaf hash for an entry. The encoding is stable
// and documented so anyone can recompute it: fields joined by '|', lines
// sorted and joined by ';', hashed with SHA-256. This is the value that gets
// anchored and that the verifier recomputes from the public entry.
func CanonicalHash(entry Entry) string {
	lines := make([]string, 0, len(entry.Lines))
	for _, line := range entry.Lines {
		lines = append(lines, fmt.Sprintf("%s:%s:%d:%s", line.AccountCode, line.Direction, line.AmountMinor, line.Currency))
	}
	sort.Strings(lines)

	canonical := strings.Join([]string{
		"v1",
		entry.ID,
		entry.InstitutionID,
		fmt.Sprintf("%d", entry.FiscalYear),
		entry.EffectiveDate,
		entry.Status,
		entry.EntryType,
		strings.Join(lines, ";"),
	}, "|")

	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

// ProofStep is one sibling on the path from a leaf to the root.
type ProofStep struct {
	Hash string `json:"hash"`
	// Left reports whether the sibling sits on the left (its hash is
	// concatenated before the running hash).
	Left bool `json:"left"`
}

// Tree is a Merkle tree over leaf hashes. Odd nodes are promoted (duplicated)
// at each level, the standard RFC 6962-style construction simplified for
// batch anchoring.
type Tree struct {
	leaves []string
	levels [][]string
}

// NewTree builds a tree from ordered leaf hashes (hex strings).
func NewTree(leaves []string) *Tree {
	tree := &Tree{leaves: leaves}
	if len(leaves) == 0 {
		return tree
	}

	level := append([]string(nil), leaves...)
	tree.levels = append(tree.levels, level)
	for len(level) > 1 {
		next := make([]string, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left // promote odd node
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, hashPair(left, right))
		}
		tree.levels = append(tree.levels, next)
		level = next
	}
	return tree
}

// Root returns the Merkle root, or "" for an empty tree.
func (t *Tree) Root() string {
	if len(t.levels) == 0 {
		return ""
	}
	return t.levels[len(t.levels)-1][0]
}

// Proof returns the inclusion proof for the leaf at index.
func (t *Tree) Proof(index int) ([]ProofStep, error) {
	if index < 0 || index >= len(t.leaves) {
		return nil, fmt.Errorf("leaf index %d out of range", index)
	}

	proof := make([]ProofStep, 0)
	position := index
	for level := 0; level < len(t.levels)-1; level++ {
		nodes := t.levels[level]
		var siblingIndex int
		var siblingLeft bool
		if position%2 == 0 {
			siblingIndex = position + 1
			siblingLeft = false
			if siblingIndex >= len(nodes) {
				siblingIndex = position // promoted odd node: sibling is itself
			}
		} else {
			siblingIndex = position - 1
			siblingLeft = true
		}
		proof = append(proof, ProofStep{Hash: nodes[siblingIndex], Left: siblingLeft})
		position /= 2
	}
	return proof, nil
}

// Verify recomputes the root from a leaf hash and its proof, and reports
// whether it matches the expected root. This is the whole trust check — it
// depends on nothing but the inputs.
func Verify(leafHash string, proof []ProofStep, expectedRoot string) bool {
	running := leafHash
	for _, step := range proof {
		if step.Left {
			running = hashPair(step.Hash, running)
		} else {
			running = hashPair(running, step.Hash)
		}
	}
	return running == expectedRoot
}

func hashPair(left, right string) string {
	sum := sha256.Sum256([]byte(left + right))
	return hex.EncodeToString(sum[:])
}
