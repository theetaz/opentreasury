-- Anchors are the tamper-evidence layer: a batch of posted journal entries is
-- hashed into a Merkle tree, and its root is committed to an append-only
-- ledger (a transparency log locally; Hyperledger Fabric in production). Each
-- entry keeps its inclusion proof so anyone can verify the entry existed,
-- unmodified, at anchor time — without trusting this database.
CREATE TABLE anchors (
  id TEXT PRIMARY KEY,
  merkle_root TEXT NOT NULL,
  entry_count INTEGER NOT NULL,
  backend TEXT NOT NULL,          -- e.g. transparency-log, fabric
  backend_ref TEXT NOT NULL,      -- tx id / sequence in the backend ledger
  anchored_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One row per anchored entry, carrying the Merkle inclusion proof.
CREATE TABLE journal_entry_anchors (
  entry_id TEXT NOT NULL REFERENCES journal_entries (id),
  anchor_id TEXT NOT NULL REFERENCES anchors (id),
  leaf_index INTEGER NOT NULL,
  leaf_hash TEXT NOT NULL,
  proof JSONB NOT NULL,           -- ordered sibling hashes with positions
  PRIMARY KEY (entry_id)
);

CREATE INDEX journal_entry_anchors_anchor_id_idx
  ON journal_entry_anchors (anchor_id);

-- Append-only transparency log: the local anchoring backend. Each row is a
-- sequential commitment of a Merkle root; row N chains prev_hash from N-1.
CREATE TABLE transparency_log (
  sequence BIGSERIAL PRIMARY KEY,
  merkle_root TEXT NOT NULL,
  prev_hash TEXT NOT NULL,
  entry_hash TEXT NOT NULL,       -- hash(sequence, merkle_root, prev_hash)
  logged_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
