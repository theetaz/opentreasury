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

export type Account = {
  code: string;
  name: string;
  accountType: "ASSET" | "LIABILITY" | "NET_WORTH" | "REVENUE" | "EXPENSE";
  parentCode?: string;
  gfsmCode?: string;
  cofogCode?: string;
  active: boolean;
  depth: number;
};

export type AccountListFilter = {
  accountType?: string;
  page?: number;
  pageSize?: number;
};

export type AccountListResult =
  | { ok: true; accounts: Account[]; pagination: PageInfo }
  | { ok: false; error: string };

export type JournalLine = {
  accountCode: string;
  direction: "DEBIT" | "CREDIT";
  amountMinor: number;
  currency: string;
};

export type JournalEntry = {
  id: string;
  institutionId: string;
  fiscalYear: number;
  effectiveDate: string;
  description: string;
  status: "POSTED" | "REVERSED";
  entryType: "STANDARD" | "REVERSAL";
  reversesEntryId?: string;
  lines: JournalLine[];
};

export type JournalEntryInput = {
  id: string;
  institutionId: string;
  fiscalYear: number;
  effectiveDate: string;
  description: string;
  lines: JournalLine[];
};

export type JournalEntryListFilter = {
  institutionId?: string;
  fiscalYear?: number;
  status?: string;
  dateFrom?: string;
  dateTo?: string;
  page?: number;
  pageSize?: number;
};

export type JournalEntryListResult =
  | { ok: true; entries: JournalEntry[]; pagination: PageInfo }
  | { ok: false; error: string };

export type Balance = {
  institutionId: string;
  accountCode: string;
  accountName?: string;
  accountType?: string;
  currency: string;
  balanceMinor: number;
};

export type BalanceListFilter = {
  institutionId?: string;
  accountCode?: string;
  page?: number;
  pageSize?: number;
};

export type BalanceListResult =
  | { ok: true; balances: Balance[]; pagination: PageInfo }
  | { ok: false; error: string };

export type HealthCheckResult =
  | { ok: true; responseTimeMs: number; checkedAt: string }
  | { ok: false; error: string; checkedAt: string };

const simulatedAccounts: Account[] = [
  { code: "1", name: "Revenue", accountType: "REVENUE", gfsmCode: "1", active: true, depth: 0 },
  { code: "11", name: "Taxes", accountType: "REVENUE", parentCode: "1", gfsmCode: "11", active: true, depth: 1 },
  { code: "114", name: "Taxes on goods and services", accountType: "REVENUE", parentCode: "11", gfsmCode: "114", active: true, depth: 2 },
  { code: "2", name: "Expense", accountType: "EXPENSE", gfsmCode: "2", active: true, depth: 0 },
  { code: "21", name: "Compensation of employees", accountType: "EXPENSE", parentCode: "2", gfsmCode: "21", active: true, depth: 1 },
  { code: "22", name: "Use of goods and services", accountType: "EXPENSE", parentCode: "2", gfsmCode: "22", active: true, depth: 1 },
  { code: "62", name: "Financial assets", accountType: "ASSET", gfsmCode: "62", active: true, depth: 0 },
  { code: "6202", name: "Currency and deposits", accountType: "ASSET", parentCode: "62", gfsmCode: "6202", active: true, depth: 1 },
  { code: "63", name: "Liabilities", accountType: "LIABILITY", gfsmCode: "63", active: true, depth: 0 },
  { code: "6", name: "Net worth", accountType: "NET_WORTH", gfsmCode: "6", active: true, depth: 0 }
];

const simulatedEntries: JournalEntry[] = [
  {
    id: "je-2026-0001",
    institutionId: "minfin",
    fiscalYear: 2026,
    effectiveDate: "2026-06-28",
    description: "Tax receipt into the treasury account",
    status: "POSTED",
    entryType: "STANDARD",
    lines: [
      { accountCode: "6202", direction: "DEBIT", amountMinor: 125000, currency: "USD" },
      { accountCode: "114", direction: "CREDIT", amountMinor: 125000, currency: "USD" }
    ]
  },
  {
    id: "je-2026-0002",
    institutionId: "minfin",
    fiscalYear: 2026,
    effectiveDate: "2026-06-29",
    description: "Office supplies purchase",
    status: "POSTED",
    entryType: "STANDARD",
    lines: [
      { accountCode: "22", direction: "DEBIT", amountMinor: 4000, currency: "USD" },
      { accountCode: "6202", direction: "CREDIT", amountMinor: 4000, currency: "USD" }
    ]
  }
];

const simulatedBalances: Balance[] = [
  { institutionId: "minfin", accountCode: "114", accountName: "Taxes on goods and services", accountType: "REVENUE", currency: "USD", balanceMinor: -125000 },
  { institutionId: "minfin", accountCode: "22", accountName: "Use of goods and services", accountType: "EXPENSE", currency: "USD", balanceMinor: 4000 },
  { institutionId: "minfin", accountCode: "6202", accountName: "Currency and deposits", accountType: "ASSET", currency: "USD", balanceMinor: 121000 }
];

import { currentAccessToken } from "@/stores/auth";

const apiBaseUrl = import.meta.env.VITE_CORE_API_URL;

/** Bearer token header when authenticated; empty in demo mode. */
function authHeaders(): Record<string, string> {
  const token = currentAccessToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

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
    const response = await fetch(`${apiBaseUrl}/healthz`, { headers: authHeaders() });
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
    const response = await fetch(`${apiBaseUrl}${buildTransactionsPath(filter)}`, { headers: authHeaders() });

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
    const response = await fetch(`${apiBaseUrl}${buildAuditEventsPath(filter)}`, { headers: authHeaders() });

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
    const response = await fetch(`${apiBaseUrl}${buildInstitutionsPath(filter)}`, { headers: authHeaders() });

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

export function buildAccountsPath(filter: AccountListFilter = {}): string {
  const params = new URLSearchParams();

  if (filter.accountType) {
    params.set("accountType", filter.accountType);
  }

  if (filter.page) {
    params.set("page", String(filter.page));
  }

  if (filter.pageSize) {
    params.set("pageSize", String(filter.pageSize));
  }

  const query = params.toString();
  return query ? `/v1/accounts?${query}` : "/v1/accounts";
}

export async function listAccounts(filter: AccountListFilter = {}): Promise<AccountListResult> {
  if (!apiBaseUrl) {
    const matches = simulatedAccounts.filter(
      (account) => !filter.accountType || account.accountType === filter.accountType
    );
    const { rows, pagination } = paginate(matches, pageWindow(filter));
    return { ok: true, accounts: rows, pagination };
  }

  try {
    const response = await fetch(`${apiBaseUrl}${buildAccountsPath(filter)}`, { headers: authHeaders() });

    if (!response.ok) {
      const body = (await response.json().catch(() => undefined)) as { error?: string } | undefined;
      return {
        ok: false,
        error: body?.error ?? `Request failed with status ${response.status}`
      };
    }

    const body = (await response.json()) as { accounts?: Account[]; pagination?: PageInfo };
    return {
      ok: true,
      accounts: body.accounts ?? [],
      pagination: body.pagination ?? fallbackPageInfo(filter, body.accounts?.length ?? 0)
    };
  } catch {
    return { ok: false, error: "Core API is not reachable" };
  }
}

export function buildJournalEntriesPath(filter: JournalEntryListFilter = {}): string {
  const params = new URLSearchParams();
  if (filter.institutionId) params.set("institutionId", filter.institutionId);
  if (filter.fiscalYear) params.set("fiscalYear", String(filter.fiscalYear));
  if (filter.status) params.set("status", filter.status);
  if (filter.dateFrom) params.set("dateFrom", filter.dateFrom);
  if (filter.dateTo) params.set("dateTo", filter.dateTo);
  if (filter.page) params.set("page", String(filter.page));
  if (filter.pageSize) params.set("pageSize", String(filter.pageSize));
  const query = params.toString();
  return query ? `/v1/journal-entries?${query}` : "/v1/journal-entries";
}

export async function listJournalEntries(filter: JournalEntryListFilter = {}): Promise<JournalEntryListResult> {
  if (!apiBaseUrl) {
    const matches = simulatedEntries
      .filter((e) => !filter.institutionId || e.institutionId === filter.institutionId)
      .filter((e) => !filter.fiscalYear || e.fiscalYear === filter.fiscalYear)
      .filter((e) => !filter.status || e.status === filter.status)
      .filter((e) => !filter.dateFrom || e.effectiveDate >= filter.dateFrom)
      .filter((e) => !filter.dateTo || e.effectiveDate <= filter.dateTo);
    const { rows, pagination } = paginate(matches, pageWindow(filter));
    return { ok: true, entries: rows, pagination };
  }

  try {
    const response = await fetch(`${apiBaseUrl}${buildJournalEntriesPath(filter)}`, { headers: authHeaders() });
    if (!response.ok) {
      const body = (await response.json().catch(() => undefined)) as { error?: string } | undefined;
      return { ok: false, error: body?.error ?? `Request failed with status ${response.status}` };
    }
    const body = (await response.json()) as { entries?: JournalEntry[]; pagination?: PageInfo };
    return {
      ok: true,
      entries: body.entries ?? [],
      pagination: body.pagination ?? fallbackPageInfo(filter, body.entries?.length ?? 0)
    };
  } catch {
    return { ok: false, error: "Core API is not reachable" };
  }
}

export async function postJournalEntry(entry: JournalEntryInput): Promise<ApiResult> {
  if (!apiBaseUrl) {
    const debits = entry.lines.filter((l) => l.direction === "DEBIT").reduce((sum, l) => sum + l.amountMinor, 0);
    const credits = entry.lines.filter((l) => l.direction === "CREDIT").reduce((sum, l) => sum + l.amountMinor, 0);
    if (entry.lines.length < 2) return { ok: false, status: 400, error: "a journal entry needs at least two lines" };
    if (debits !== credits) return { ok: false, status: 400, error: "entry debits and credits are not balanced per currency" };
    return { ok: true, status: 201, data: { id: entry.id } };
  }

  try {
    const response = await fetch(`${apiBaseUrl}/v1/journal-entries`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify(entry)
    });
    if (!response.ok) {
      const body = (await response.json().catch(() => undefined)) as { error?: string } | undefined;
      return { ok: false, status: response.status, error: body?.error ?? `Request failed with status ${response.status}` };
    }
    return { ok: true, status: response.status, data: await response.json().catch(() => undefined) };
  } catch {
    return { ok: false, status: 0, error: "Core API is not reachable" };
  }
}

export function buildBalancesPath(filter: BalanceListFilter = {}): string {
  const params = new URLSearchParams();
  if (filter.institutionId) params.set("institutionId", filter.institutionId);
  if (filter.accountCode) params.set("accountCode", filter.accountCode);
  if (filter.page) params.set("page", String(filter.page));
  if (filter.pageSize) params.set("pageSize", String(filter.pageSize));
  const query = params.toString();
  return query ? `/v1/balances?${query}` : "/v1/balances";
}

export async function listBalances(filter: BalanceListFilter = {}): Promise<BalanceListResult> {
  if (!apiBaseUrl) {
    const matches = simulatedBalances
      .filter((b) => !filter.institutionId || b.institutionId === filter.institutionId)
      .filter((b) => !filter.accountCode || b.accountCode === filter.accountCode);
    const { rows, pagination } = paginate(matches, pageWindow(filter));
    return { ok: true, balances: rows, pagination };
  }

  try {
    const response = await fetch(`${apiBaseUrl}${buildBalancesPath(filter)}`, { headers: authHeaders() });
    if (!response.ok) {
      const body = (await response.json().catch(() => undefined)) as { error?: string } | undefined;
      return { ok: false, error: body?.error ?? `Request failed with status ${response.status}` };
    }
    const body = (await response.json()) as { balances?: Balance[]; pagination?: PageInfo };
    return {
      ok: true,
      balances: body.balances ?? [],
      pagination: body.pagination ?? fallbackPageInfo(filter, body.balances?.length ?? 0)
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
        "Content-Type": "application/json",
        ...authHeaders()
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
    pageSize: filter.pageSize ?? filter.limit ?? 15
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
