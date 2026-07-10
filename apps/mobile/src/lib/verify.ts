// Independent proof verification, mirroring the Go implementation in
// services/ledger-gateway/internal/merkle byte for byte. The app recomputes
// the canonical entry hash and walks the Merkle inclusion proof itself —
// trusting nothing the server stored beyond the anchored root.

export interface PublicEntryLine {
  accountCode: string;
  direction: string;
  amountMinor: number;
  currency: string;
}

export interface PublicEntry {
  id: string;
  institutionId: string;
  fiscalYear: number;
  effectiveDate: string;
  status: string;
  entryType: string;
  lines: PublicEntryLine[];
}

export interface ProofStep {
  hash: string;
  left: boolean;
}

export interface AnchorReceipt {
  id: string;
  merkleRoot: string;
  backend: string;
  backendRef: string;
  anchoredAt: string;
}

export interface ProofResponse {
  entry: PublicEntry;
  anchor: AnchorReceipt;
  leafHash: string;
  proof: ProofStep[];
}

/** SHA-256 as lowercase hex; injected so tests use node and the app expo-crypto. */
export type Sha256Hex = (value: string) => Promise<string>;

// Canonical encoding v1 (documented in ADR-0004): fields joined by '|',
// lines formatted acct:dir:amount:currency, sorted lexically, joined by ';'.
export function canonicalEntryString(entry: PublicEntry): string {
  const lines = entry.lines
    .map(
      (line) =>
        `${line.accountCode}:${line.direction}:${line.amountMinor}:${line.currency}`,
    )
    .sort();

  return [
    "v1",
    entry.id,
    entry.institutionId,
    String(entry.fiscalYear),
    entry.effectiveDate,
    entry.status,
    entry.entryType,
    lines.join(";"),
  ].join("|");
}

export interface VerificationResult {
  verified: boolean;
  recomputedLeaf: string;
  recomputedRoot: string;
}

export async function verifyProof(
  response: ProofResponse,
  sha256Hex: Sha256Hex,
): Promise<VerificationResult> {
  const recomputedLeaf = await sha256Hex(canonicalEntryString(response.entry));

  let running = recomputedLeaf;
  for (const step of response.proof) {
    running = step.left
      ? await sha256Hex(step.hash + running)
      : await sha256Hex(running + step.hash);
  }

  return {
    verified:
      recomputedLeaf === response.leafHash &&
      running === response.anchor.merkleRoot,
    recomputedLeaf,
    recomputedRoot: running,
  };
}
