import { TreasuryTransaction } from "./api";

export type TransactionFormState = {
  id: string;
  institutionId: string;
  fiscalYear: string;
  amountMinor: string;
  currency: string;
  transactionDate: string;
  description: string;
};

export function toTransactionPayload(form: TransactionFormState): TreasuryTransaction {
  return {
    id: form.id.trim(),
    institutionId: form.institutionId.trim(),
    fiscalYear: Number(form.fiscalYear),
    amountMinor: Number(form.amountMinor),
    currency: form.currency.trim().toUpperCase(),
    transactionDate: form.transactionDate,
    description: form.description.trim()
  };
}
