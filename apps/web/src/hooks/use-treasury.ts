import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  checkCoreApiHealth,
  createTransaction,
  listAccounts,
  listAuditEvents,
  listInstitutions,
  listTransactions,
  validateTransaction,
  type AccountListFilter,
  type AuditEventListFilter,
  type InstitutionListFilter,
  type TransactionListFilter,
  type TreasuryTransaction
} from "@/api";

export function useAccounts(filter: AccountListFilter) {
  return useQuery({
    queryKey: ["accounts", filter],
    queryFn: () => listAccounts(filter)
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
