import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  checkCoreApiHealth,
  createTransaction,
  listAccounts,
  listAuditEvents,
  listBalances,
  listInstitutions,
  createCommitment,
  listCommitments,
  listJournalEntries,
  listReconciliation,
  listStagingRecords,
  listTransactions,
  postJournalEntry,
  validateTransaction,
  type AccountListFilter,
  type AuditEventListFilter,
  type BalanceListFilter,
  type InstitutionListFilter,
  type JournalEntryInput,
  type CommitmentInput,
  type CommitmentListFilter,
  type JournalEntryListFilter,
  type ReconciliationListFilter,
  type StagingListFilter,
  type TransactionListFilter,
  type TreasuryTransaction
} from "@/api";

export function useAccounts(filter: AccountListFilter) {
  return useQuery({
    queryKey: ["accounts", filter],
    queryFn: () => listAccounts(filter)
  });
}

export function useJournalEntries(filter: JournalEntryListFilter) {
  return useQuery({
    queryKey: ["journal-entries", filter],
    queryFn: () => listJournalEntries(filter)
  });
}

export function useBalances(filter: BalanceListFilter) {
  return useQuery({
    queryKey: ["balances", filter],
    queryFn: () => listBalances(filter)
  });
}

export function useStagingRecords(filter: StagingListFilter) {
  return useQuery({
    queryKey: ["staging-records", filter],
    queryFn: () => listStagingRecords(filter)
  });
}

export function useCommitments(filter: CommitmentListFilter) {
  return useQuery({
    queryKey: ["commitments", filter],
    queryFn: () => listCommitments(filter)
  });
}

export function useCreateCommitment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CommitmentInput) => createCommitment(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["commitments"] });
    }
  });
}

export function useReconciliation(filter: ReconciliationListFilter) {
  return useQuery({
    queryKey: ["reconciliation", filter],
    queryFn: () => listReconciliation(filter)
  });
}

export function usePostJournalEntry() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (entry: JournalEntryInput) => postJournalEntry(entry),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["journal-entries"] });
      void queryClient.invalidateQueries({ queryKey: ["balances"] });
      void queryClient.invalidateQueries({ queryKey: ["audit-events"] });
    }
  });
}

export function useHealth() {
  return useQuery({
    queryKey: ["health"],
    queryFn: checkCoreApiHealth,
    refetchInterval: 30_000
  });
}

export function useTransactions(filter: TransactionListFilter = {}) {
  return useQuery({
    queryKey: ["transactions", filter],
    queryFn: () => listTransactions(filter)
  });
}

export function useAuditEvents(filter: AuditEventListFilter = {}) {
  return useQuery({
    queryKey: ["audit-events", filter],
    queryFn: () => listAuditEvents(filter)
  });
}

// Full institution set for dropdown filters and relationship pickers.
export function useInstitutions() {
  return useQuery({
    queryKey: ["institutions", "all"],
    queryFn: () => listInstitutions({ pageSize: 100 }),
    staleTime: 5 * 60_000
  });
}

// Paged institution browsing for the institutions table.
export function useInstitutionsPage(filter: InstitutionListFilter) {
  return useQuery({
    queryKey: ["institutions", filter],
    queryFn: () => listInstitutions(filter)
  });
}

export function useValidateTransaction() {
  return useMutation({
    mutationFn: (tx: TreasuryTransaction) => validateTransaction(tx)
  });
}

export function useCreateTransaction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (tx: TreasuryTransaction) => createTransaction(tx),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["transactions"] });
      void queryClient.invalidateQueries({ queryKey: ["audit-events"] });
    }
  });
}
