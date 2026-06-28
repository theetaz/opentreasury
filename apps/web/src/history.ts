import type { TreasuryTransaction, TransactionListFilter } from "./api";

export type RecentEvent = {
  id: string;
  institution: string;
  date: string;
  amount: string;
  status: "created" | "blocked";
  time: string;
};

export function buildTransactionsPath(filter: TransactionListFilter = {}): string {
  const params = new URLSearchParams();

  if (filter.institutionId) {
    params.set("institutionId", filter.institutionId);
  }

  if (filter.fiscalYear) {
    params.set("fiscalYear", String(filter.fiscalYear));
  }

  if (filter.limit) {
    params.set("limit", String(filter.limit));
  }

  const query = params.toString();
  return query ? `/v1/transactions?${query}` : "/v1/transactions";
}

export function formatTransactionAmount(currency: string, amountMinor: number): string {
  return `${currency} ${(amountMinor / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  })}`;
}

export function toRecentEvent(transaction: TreasuryTransaction): RecentEvent {
  return {
    id: transaction.id,
    institution: transaction.institutionId,
    date: transaction.transactionDate,
    amount: formatTransactionAmount(transaction.currency, transaction.amountMinor),
    status: "created",
    time: "Recorded"
  };
}
