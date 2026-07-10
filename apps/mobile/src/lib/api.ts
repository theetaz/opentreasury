// Client for the anonymous public tier (/public/v1). The mobile app is the
// public-monitoring persona: it only ever reads published, redacted data —
// no authentication, no privileged surface.
import { buildQuery } from "./format";
import type { ProofResponse, PublicEntry } from "./verify";

const baseUrl =
  process.env.EXPO_PUBLIC_API_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

export interface Pagination {
  page: number;
  pageSize: number;
  total: number;
}

export interface Institution {
  id: string;
  name: string;
  type: string;
  countryCode: string;
  status: string;
}

export interface Balance {
  institutionId: string;
  accountCode: string;
  accountName: string;
  accountType: string;
  currency: string;
  balanceMinor: number;
}

export interface EntriesFilter {
  page?: number;
  pageSize?: number;
  institutionId?: string;
  fiscalYear?: number;
}

async function get<T>(path: string): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`);
  if (!response.ok) {
    throw new Error(`public API responded ${response.status}`);
  }
  return (await response.json()) as T;
}

export const publicApi = {
  institutions: (page = 1, pageSize = 50) =>
    get<{ institutions: Institution[]; pagination: Pagination }>(
      `/public/v1/institutions${buildQuery({ page, pageSize })}`,
    ),

  balances: (institutionId?: string, page = 1, pageSize = 50) =>
    get<{ balances: Balance[]; pagination: Pagination }>(
      `/public/v1/balances${buildQuery({ institutionId, page, pageSize })}`,
    ),

  journalEntries: (filter: EntriesFilter = {}) =>
    get<{ entries: PublicEntry[]; pagination: Pagination }>(
      `/public/v1/journal-entries${buildQuery({
        page: filter.page ?? 1,
        pageSize: filter.pageSize ?? 15,
        institutionId: filter.institutionId,
        fiscalYear: filter.fiscalYear,
      })}`,
    ),

  entryProof: (id: string) =>
    get<ProofResponse>(`/public/v1/entries/${encodeURIComponent(id)}/proof`),
};
