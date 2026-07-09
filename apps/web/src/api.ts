import { buildAuditEventsPath } from "./audit";
import { buildTransactionsPath } from "./history";
import { buildInstitutionsPath } from "./institutions";

export type TreasuryTransaction = {
  id: string;
  institutionId: string;
  fiscalYear: number;
  amountMinor: number;
  currency: string;
  description: string;
  transactionDate: string;
};

export type PageInfo = {
  page: number;
  pageSize: number;
  total: number;
};

export type TransactionListFilter = {
  institutionId?: string;
  fiscalYear?: number;
  dateFrom?: string;
  dateTo?: string;
  amountGte?: number;
  amountLte?: number;
  page?: number;
  pageSize?: number;
  limit?: number;
};

export type AuditEventListFilter = {
  institutionId?: string;
  dateFrom?: string;
  dateTo?: string;
  page?: number;
  pageSize?: number;
  limit?: number;
};

export type InstitutionListFilter = {
  status?: string;
  page?: number;
  pageSize?: number;
  limit?: number;
};

export type ApiResult =
  | { ok: true; status: number; data?: unknown }
  | { ok: false; status: number; error: string };

export type ValidationResponse = {
  status: "VALID";
  code: "VALIDATION_SUCCESS";
  message: string;
  transactionId: string;
  institutionId: string;
  fiscalYear: number;
  amountMinor: number;
  currency: string;
  transactionDate: string;
};

export type TransactionListResult =
  | { ok: true; transactions: TreasuryTransaction[]; pagination: PageInfo }
  | { ok: false; error: string };

export type AuditEvent = {
  id: string;
  eventType: "TRANSACTION_CREATED";
  transactionId: string;
  institutionId: string;
  occurredAt: string;
  summary: string;
};

export type AuditEventListResult =
  | { ok: true; events: AuditEvent[]; pagination: PageInfo }
  | { ok: false; error: string };

export type Institution = {
  id: string;
  name: string;
  type: "MINISTRY" | "DEPARTMENT" | "AGENCY" | "COUNTY";
  countryCode: string;
  status: "ACTIVE" | "INACTIVE";
};

export type InstitutionListResult =
  | { ok: true; institutions: Institution[]; pagination: PageInfo }
  | { ok: false; error: string };

export type HealthCheckResult =
  | { ok: true; responseTimeMs: number; checkedAt: string }
  | { ok: false; error: string; checkedAt: string };

const apiBaseUrl = import.meta.env.VITE_CORE_API_URL;

const simulatedTransactions: TreasuryTransaction[] = [
  {
    id: "txn-2026-0001",
    institutionId: "minfin",
    fiscalYear: 2026,
    amountMinor: 125000,
    currency: "USD",
    description: "Road maintenance payment",
    transactionDate: "2026-06-28"
  },
  {
    id: "txn-2026-0000",
    institutionId: "transport",
    fiscalYear: 2026,
    amountMinor: 98000,
    currency: "USD",
    description: "Bridge inspection payment",
    transactionDate: "2026-06-27"
  },
  {
    id: "txn-2025-0942",
    institutionId: "health",
    fiscalYear: 2025,
    amountMinor: 450000,
    currency: "USD",
    description: "Clinic equipment procurement",
    transactionDate: "2025-12-18"
  }
];

const simulatedAuditEvents: AuditEvent[] = [
  {
    id: "audit-txn-2026-0001-created",
    eventType: "TRANSACTION_CREATED",
    transactionId: "txn-2026-0001",
    institutionId: "minfin",
    occurredAt: "2026-06-28T10:24:28Z",
    summary: "Transaction txn-2026-0001 was created."
  },
  {
    id: "audit-txn-2026-0000-created",
    eventType: "TRANSACTION_CREATED",
    transactionId: "txn-2026-0000",
    institutionId: "transport",
    occurredAt: "2026-06-27T15:10:42Z",
    summary: "Transaction txn-2026-0000 was created."
  },
  {
    id: "audit-txn-2025-0942-created",
    eventType: "TRANSACTION_CREATED",
    transactionId: "txn-2025-0942",
    institutionId: "health",
    occurredAt: "2025-12-18T09:45:00Z",
    summary: "Transaction txn-2025-0942 was created."
  }
];

const simulatedInstitutions: Institution[] = [
  {
    id: "minfin",
    name: "Ministry of Finance",
    type: "MINISTRY",
    countryCode: "KE",
    status: "ACTIVE"
  },
  {
    id: "health",
    name: "Ministry of Health",
    type: "MINISTRY",
    countryCode: "KE",
    status: "ACTIVE"
  },
  {
    id: "transport",
    name: "Transport Infrastructure Agency",
    type: "AGENCY",
    countryCode: "KE",
    status: "ACTIVE"
  }
];

export async function checkCoreApiHealth(): Promise<HealthCheckResult> {
  if (!apiBaseUrl) {
    return { ok: true, responseTimeMs: 0, checkedAt: new Date().toISOString() };
  }

  const startedAt = Date.now();
  try {
    const response = await fetch(`${apiBaseUrl}/healthz`);
    if (!response.ok) {
      return {
        ok: false,
        error: `Health check failed with status ${response.status}`,
        checkedAt: new Date().toISOString()
      };
    }

    return {
      ok: true,
      responseTimeMs: Date.now() - startedAt,
      checkedAt: new Date().toISOString()
    };
  } catch {
    return {
      ok: false,
      error: "Core API is not reachable",
      checkedAt: new Date().toISOString()
    };
  }
}

export async function listTransactions(filter: TransactionListFilter = { limit: 5 }): Promise<TransactionListResult> {
  if (!apiBaseUrl) {
    return simulateTransactionList(filter);
  }

  try {
    const response = await fetch(`${apiBaseUrl}${buildTransactionsPath(filter)}`);

    if (!response.ok) {
      const body = (await response.json().catch(() => undefined)) as { error?: string } | undefined;
      return {
        ok: false,
        error: body?.error ?? `Request failed with status ${response.status}`
      };
    }

    const body = (await response.json()) as { transactions?: TreasuryTransaction[]; pagination?: PageInfo };
    return {
      ok: true,
      transactions: body.transactions ?? [],
      pagination: body.pagination ?? fallbackPageInfo(filter, body.transactions?.length ?? 0)
    };
  } catch {
    return { ok: false, error: "Core API is not reachable" };
  }
}

export async function listAuditEvents(filter: AuditEventListFilter = { limit: 5 }): Promise<AuditEventListResult> {
  if (!apiBaseUrl) {
    return simulateAuditEventList(filter);
  }

  try {
    const response = await fetch(`${apiBaseUrl}${buildAuditEventsPath(filter)}`);

    if (!response.ok) {
      const body = (await response.json().catch(() => undefined)) as { error?: string } | undefined;
      return {
        ok: false,
        error: body?.error ?? `Request failed with status ${response.status}`
      };
    }

    const body = (await response.json()) as { events?: AuditEvent[]; pagination?: PageInfo };
    return {
      ok: true,
      events: body.events ?? [],
      pagination: body.pagination ?? fallbackPageInfo(filter, body.events?.length ?? 0)
    };
  } catch {
    return { ok: false, error: "Core API is not reachable" };
  }
}

export async function listInstitutions(filter: InstitutionListFilter = { limit: 5 }): Promise<InstitutionListResult> {
  if (!apiBaseUrl) {
    return simulateInstitutionList(filter);
  }

  try {
    const response = await fetch(`${apiBaseUrl}${buildInstitutionsPath(filter)}`);

    if (!response.ok) {
      const body = (await response.json().catch(() => undefined)) as { error?: string } | undefined;
      return {
        ok: false,
        error: body?.error ?? `Request failed with status ${response.status}`
      };
    }

    const body = (await response.json()) as { institutions?: Institution[]; pagination?: PageInfo };
    return {
      ok: true,
      institutions: body.institutions ?? [],
      pagination: body.pagination ?? fallbackPageInfo(filter, body.institutions?.length ?? 0)
    };
  } catch {
    return { ok: false, error: "Core API is not reachable" };
  }
}

export async function validateTransaction(transaction: TreasuryTransaction): Promise<ApiResult> {
  if (!apiBaseUrl) {
    return simulateValidation(transaction);
  }

  return postTransaction("/v1/transactions/validate", transaction);
}

export async function createTransaction(transaction: TreasuryTransaction): Promise<ApiResult> {
  if (!apiBaseUrl) {
    const validation = simulateValidation(transaction);
    return validation.ok ? { ok: true, status: 201, data: { id: transaction.id } } : validation;
  }

  return postTransaction("/v1/transactions", transaction);
}

async function postTransaction(path: string, transaction: TreasuryTransaction): Promise<ApiResult> {
  try {
    const response = await fetch(`${apiBaseUrl}${path}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(transaction)
    });

    if (response.ok) {
      if (response.status === 204) {
        return { ok: true, status: response.status };
      }

      return { ok: true, status: response.status, data: await response.json() };
    }

    const body = (await response.json().catch(() => undefined)) as { error?: string } | undefined;
    return {
      ok: false,
      status: response.status,
      error: body?.error ?? `Request failed with status ${response.status}`
    };
  } catch {
    return {
      ok: false,
      status: 0,
      error: "Core API is not reachable"
    };
  }
}

type PageWindow = { page: number; pageSize: number };

function pageWindow(filter: { page?: number; pageSize?: number; limit?: number }): PageWindow {
  return {
    page: filter.page && filter.page > 0 ? filter.page : 1,
    pageSize: filter.pageSize ?? filter.limit ?? 25
  };
}

function paginate<T>(rows: T[], window: PageWindow): { rows: T[]; pagination: PageInfo } {
  const start = (window.page - 1) * window.pageSize;
  return {
    rows: rows.slice(start, start + window.pageSize),
    pagination: { page: window.page, pageSize: window.pageSize, total: rows.length }
  };
}

function fallbackPageInfo(filter: { page?: number; pageSize?: number; limit?: number }, count: number): PageInfo {
  const window = pageWindow(filter);
  return { page: window.page, pageSize: window.pageSize, total: count };
}

function simulateTransactionList(filter: TransactionListFilter): TransactionListResult {
  const matches = simulatedTransactions
    .filter((transaction) => !filter.institutionId || transaction.institutionId === filter.institutionId)
    .filter((transaction) => !filter.fiscalYear || transaction.fiscalYear === filter.fiscalYear)
    .filter((transaction) => !filter.dateFrom || transaction.transactionDate >= filter.dateFrom)
    .filter((transaction) => !filter.dateTo || transaction.transactionDate <= filter.dateTo)
    .filter((transaction) => !filter.amountGte || transaction.amountMinor >= filter.amountGte)
    .filter((transaction) => !filter.amountLte || transaction.amountMinor <= filter.amountLte);
  const { rows, pagination } = paginate(matches, pageWindow(filter));
  return { ok: true, transactions: rows, pagination };
}

function simulateAuditEventList(filter: AuditEventListFilter): AuditEventListResult {
  const matches = simulatedAuditEvents
    .filter((event) => !filter.institutionId || event.institutionId === filter.institutionId)
    .filter((event) => !filter.dateFrom || event.occurredAt.slice(0, 10) >= filter.dateFrom)
    .filter((event) => !filter.dateTo || event.occurredAt.slice(0, 10) <= filter.dateTo);
  const { rows, pagination } = paginate(matches, pageWindow(filter));
  return { ok: true, events: rows, pagination };
}

function simulateInstitutionList(filter: InstitutionListFilter): InstitutionListResult {
  const matches = simulatedInstitutions.filter(
    (institution) => !filter.status || institution.status === filter.status
  );
  const { rows, pagination } = paginate(matches, pageWindow(filter));
  return { ok: true, institutions: rows, pagination };
}

function simulateValidation(transaction: TreasuryTransaction): ApiResult {
  if (!transaction.id) {
    return { ok: false, status: 400, error: "missing transaction id" };
  }

  if (!transaction.institutionId) {
    return { ok: false, status: 400, error: "missing institution id" };
  }

  if (transaction.fiscalYear <= 0 || Number.isNaN(transaction.fiscalYear)) {
    return { ok: false, status: 400, error: "invalid fiscal year" };
  }

  if (transaction.amountMinor <= 0 || Number.isNaN(transaction.amountMinor)) {
    return { ok: false, status: 400, error: "invalid amount" };
  }

  if (!/^[A-Z]{3}$/.test(transaction.currency)) {
    return { ok: false, status: 400, error: "invalid currency" };
  }

  if (!transaction.description) {
    return { ok: false, status: 400, error: "missing description" };
  }

  if (!transaction.transactionDate) {
    return { ok: false, status: 400, error: "missing transaction date" };
  }

  if (Number.isNaN(Date.parse(transaction.transactionDate))) {
    return { ok: false, status: 400, error: "invalid transaction date" };
  }

  return {
    ok: true,
    status: 200,
    data: {
      status: "VALID",
      code: "VALIDATION_SUCCESS",
      message: "Transaction is valid.",
      transactionId: transaction.id,
      institutionId: transaction.institutionId,
      fiscalYear: transaction.fiscalYear,
      amountMinor: transaction.amountMinor,
      currency: transaction.currency,
      transactionDate: transaction.transactionDate
    } satisfies ValidationResponse
  };
}
