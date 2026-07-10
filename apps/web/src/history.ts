import type { TreasuryTransaction, TransactionListFilter } from "./api";

export type RecentEvent = {
  id: string;
  institution: string;
  date: string;
  amount: string;
  status: "created" | "blocked";
  time: string;
};

export type HistoryFilterForm = {
  institutionId: string;
  fiscalYear: string;
  limit: string;
};

export function buildTransactionsPath(filter: TransactionListFilter = {}): string {
  const params = new URLSearchParams();

  if (filter.institutionId) {
    params.set("institutionId", filter.institutionId);
  }

  if (filter.fiscalYear) {
    params.set("fiscalYear", String(filter.fiscalYear));
  }

  if (filter.status) {
    params.set("status", filter.status);
  }

  if (filter.dateFrom) {
    params.set("dateFrom", filter.dateFrom);
  }

  if (filter.dateTo) {
    params.set("dateTo", filter.dateTo);
  }

  if (filter.amountGte) {
    params.set("amountGte", String(filter.amountGte));
  }

  if (filter.amountLte) {
    params.set("amountLte", String(filter.amountLte));
  }

  if (filter.page) {
    params.set("page", String(filter.page));
  }

  if (filter.pageSize) {
    params.set("pageSize", String(filter.pageSize));
  }

  if (filter.limit && !filter.pageSize) {
    params.set("limit", String(filter.limit));
  }

  const query = params.toString();
  return query ? `/v1/transactions?${query}` : "/v1/transactions";
}

export function toTransactionListFilter(form: HistoryFilterForm): TransactionListFilter {
  const filter: TransactionListFilter = {};
  const institutionId = form.institutionId.trim();
  const fiscalYear = Number(form.fiscalYear);
  const limit = Number(form.limit);

  if (institutionId) {
    filter.institutionId = institutionId;
  }

  if (Number.isInteger(fiscalYear) && fiscalYear > 0) {
    filter.fiscalYear = fiscalYear;
  }

  if (Number.isInteger(limit) && limit > 0) {
    // The API rejects limits above 100 (ADR-0002); cap in the form layer.
    filter.limit = Math.min(limit, 100);
  }

  return filter;
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
