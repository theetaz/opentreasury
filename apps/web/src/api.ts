export type TreasuryTransaction = {
  id: string;
  institutionId: string;
  fiscalYear: number;
  amountMinor: number;
  currency: string;
  description: string;
  transactionDate: string;
};

export type ApiResult =
  | { ok: true; status: number; data?: unknown }
  | { ok: false; status: number; error: string };

const apiBaseUrl = import.meta.env.VITE_CORE_API_URL;

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

  return { ok: true, status: 204 };
}
