import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  checkCoreApiHealth,
  createTransaction,
  listAuditEvents,
  listInstitutions,
  listTransactions,
  validateTransaction,
  type AuditEventListFilter,
  type TransactionListFilter,
  type TreasuryTransaction
} from "@/api";

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

export function useInstitutions() {
  return useQuery({
    queryKey: ["institutions"],
    queryFn: () => listInstitutions({})
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
