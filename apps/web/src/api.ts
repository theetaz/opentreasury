import { buildTransactionsPath } from "./history";

export type TreasuryTransaction = {
  id: string;
  institutionId: string;
  fiscalYear: number;
  amountMinor: number;
  currency: string;
  description: string;
  transactionDate: string;
};

export type TransactionListFilter = {
  institutionId?: string;
  fiscalYear?: number;
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
  | { ok: true; transactions: TreasuryTransaction[] }
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
    return { ok: true, transactions: filterSimulatedTransactions(filter) };
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

    const body = (await response.json()) as { transactions?: TreasuryTransaction[] };
    return { ok: true, transactions: body.transactions ?? [] };
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

function filterSimulatedTransactions(filter: TransactionListFilter): TreasuryTransaction[] {
  return simulatedTransactions
    .filter((transaction) => !filter.institutionId || transaction.institutionId === filter.institutionId)
    .filter((transaction) => !filter.fiscalYear || transaction.fiscalYear === filter.fiscalYear)
    .slice(0, filter.limit ?? 5);
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
