import { createHash } from "node:crypto";

import { canonicalEntryString, verifyProof, type ProofResponse } from "./verify";

// Tests inject node's SHA-256; the app injects expo-crypto's. Both hash the
// same canonical strings, so this cross-checks the encoding, not the digest.
const sha256Hex = async (value: string) =>
  createHash("sha256").update(value).digest("hex");

// Real vector produced by the Go gateway and anchored on the local Fabric
// network: entry je-fabric-1 hashed to this exact leaf. If the TypeScript
// canonical encoding ever drifts from Go's, this fails.
const entry = {
  id: "je-fabric-1",
  institutionId: "minfin",
  fiscalYear: 2026,
  effectiveDate: "2026-07-10",
  status: "POSTED",
  entryType: "STANDARD",
  lines: [
    { accountCode: "6202", direction: "DEBIT", amountMinor: 50000, currency: "USD" },
    { accountCode: "114", direction: "CREDIT", amountMinor: 50000, currency: "USD" },
  ],
};

const GO_LEAF_HASH = "5a390e17ba45b553647b6c56fe5fd5e464908b1af2ea6ab26f91a4372185326a";

const anchorWithRoot = (merkleRoot: string) => ({
  id: "anc-test",
  merkleRoot,
  backend: "fabric",
  backendRef: "fabric:opentreasury:tx",
  anchoredAt: "2026-07-10T07:28:20Z",
});

test("canonical encoding matches the Go implementation byte for byte", async () => {
  expect(canonicalEntryString(entry)).toBe(
    "v1|je-fabric-1|minfin|2026|2026-07-10|POSTED|STANDARD|" +
      "114:CREDIT:50000:USD;6202:DEBIT:50000:USD",
  );
  expect(await sha256Hex(canonicalEntryString(entry))).toBe(GO_LEAF_HASH);
});

test("line order does not change the canonical string", () => {
  const swapped = { ...entry, lines: [entry.lines[1], entry.lines[0]] };
  expect(canonicalEntryString(swapped)).toBe(canonicalEntryString(entry));
});

test("a single-leaf proof verifies against the anchored root", async () => {
  // je-fabric-1 was a batch of one: leaf === root, empty proof path.
  const proof: ProofResponse = {
    entry,
    anchor: anchorWithRoot(GO_LEAF_HASH),
    leafHash: GO_LEAF_HASH,
    proof: [],
  };
  const result = await verifyProof(proof, sha256Hex);
  expect(result.verified).toBe(true);
});

test("tampering with an amount makes verification fail", async () => {
  const tampered: ProofResponse = {
    entry: {
      ...entry,
      lines: entry.lines.map((line) => ({ ...line, amountMinor: 50001 })),
    },
    anchor: anchorWithRoot(GO_LEAF_HASH),
    leafHash: GO_LEAF_HASH,
    proof: [],
  };
  const result = await verifyProof(tampered, sha256Hex);
  expect(result.verified).toBe(false);
});

test("a multi-step proof walks left and right siblings correctly", async () => {
  // Build a two-leaf tree by hand: root = H(leafA + leafB).
  const leafA = await sha256Hex(canonicalEntryString(entry));
  const leafB = await sha256Hex("sibling");
  const root = await sha256Hex(leafA + leafB);

  const good = await verifyProof(
    { entry, anchor: anchorWithRoot(root), leafHash: leafA, proof: [{ hash: leafB, left: false }] },
    sha256Hex,
  );
  expect(good.verified).toBe(true);

  const wrongSide = await verifyProof(
    { entry, anchor: anchorWithRoot(root), leafHash: leafA, proof: [{ hash: leafB, left: true }] },
    sha256Hex,
  );
  expect(wrongSide.verified).toBe(false);
});
