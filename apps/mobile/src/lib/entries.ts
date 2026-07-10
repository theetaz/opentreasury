import type { PublicEntry } from "./verify";

/** Total debit amount and currency of an entry (entries balance per currency). */
export function entryTotal(entry: PublicEntry): [number, string] {
  let total = 0;
  let currency = "";
  for (const line of entry.lines ?? []) {
    if (line.direction === "DEBIT") {
      total += line.amountMinor;
      currency = line.currency;
    }
  }
  return [total, currency || "USD"];
}
