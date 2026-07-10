import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { CheckCircle2, ShieldCheck, ShieldX } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from "@/components/ui/dialog";
import { fetchEntryProof, type EntryProof } from "@/api";

// Recompute the canonical leaf hash in the browser — trusting nothing the
// server sent but the entry fields themselves.
async function canonicalHash(entry: EntryProof["entry"]): Promise<string> {
  const lines = entry.lines
    .map((line) => `${line.accountCode}:${line.direction}:${line.amountMinor}:${line.currency}`)
    .sort()
    .join(";");
  const canonical = [
    "v1",
    entry.id,
    entry.institutionId,
    String(entry.fiscalYear),
    entry.effectiveDate,
    entry.status,
    entry.entryType,
    lines
  ].join("|");
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(canonical));
  return [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

async function hashPair(left: string, right: string): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(left + right));
  return [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

type VerifyResult =
  | { state: "verified"; leaf: string; root: string; backend: string; anchoredAt: string }
  | { state: "failed"; reason: string }
  | { state: "unanchored" };

async function verify(entryId: string): Promise<VerifyResult> {
  const result = await fetchEntryProof(entryId);
  if (!result.ok) {
    if (result.status === 404) return { state: "unanchored" };
    return { state: "failed", reason: result.error };
  }

  const { entry, anchor, proof } = result.proof;
  const recomputed = await canonicalHash(entry);
  if (recomputed !== result.proof.leafHash) {
    return { state: "failed", reason: "The entry does not hash to the anchored leaf." };
  }

  let running = recomputed;
  for (const step of proof) {
    running = step.left ? await hashPair(step.hash, running) : await hashPair(running, step.hash);
  }
  if (running !== anchor.merkleRoot) {
    return { state: "failed", reason: "The inclusion proof does not reach the anchored root." };
  }

  return { state: "verified", leaf: recomputed, root: anchor.merkleRoot, backend: anchor.backend, anchoredAt: anchor.anchoredAt };
}

export function VerifyEntry({ entryId }: { entryId: string }) {
  const [open, setOpen] = useState(false);
  const mutation = useMutation({ mutationFn: () => verify(entryId) });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm" className="gap-1.5 text-primary" onClick={() => mutation.mutate()}>
          <ShieldCheck aria-hidden className="size-4" />
          Verify
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Verify entry {entryId}</DialogTitle>
          <DialogDescription>
            Recomputed in your browser from the entry and its Merkle proof — independent of our servers.
          </DialogDescription>
        </DialogHeader>
        {mutation.isPending ? (
          <p className="text-sm text-muted-foreground">Fetching proof and recomputing…</p>
        ) : mutation.data?.state === "verified" ? (
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-success">
              <CheckCircle2 aria-hidden className="size-5" />
              <span className="font-semibold">Verified — anchored, unmodified.</span>
            </div>
            <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
              <dt className="text-muted-foreground">Leaf hash</dt>
              <dd className="truncate font-mono">{mutation.data.leaf}</dd>
              <dt className="text-muted-foreground">Merkle root</dt>
              <dd className="truncate font-mono">{mutation.data.root}</dd>
              <dt className="text-muted-foreground">Backend</dt>
              <dd className="font-mono">{mutation.data.backend}</dd>
              <dt className="text-muted-foreground">Anchored</dt>
              <dd className="figures-tabular">{mutation.data.anchoredAt}</dd>
            </dl>
          </div>
        ) : mutation.data?.state === "unanchored" ? (
          <p className="text-sm text-muted-foreground">
            This entry has not been anchored yet — the ledger gateway anchors posted entries in batches.
          </p>
        ) : mutation.data?.state === "failed" ? (
          <div className="flex items-start gap-2 text-destructive">
            <ShieldX aria-hidden className="mt-0.5 size-5" />
            <div>
              <p className="font-semibold">Verification failed.</p>
              <p className="text-sm">{mutation.data.reason}</p>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
