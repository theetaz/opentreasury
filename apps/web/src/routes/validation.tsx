import { useState } from "react";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useCreateTransaction, useValidateTransaction } from "@/hooks/use-treasury";
import { toTransactionPayload, type TransactionFormState } from "@/transaction";
import type { ApiResult } from "@/api";

const initialForm: TransactionFormState = {
  id: "txn-2026-0001",
  institutionId: "minfin",
  fiscalYear: "2026",
  amountMinor: "125000",
  currency: "USD",
  description: "Road maintenance payment",
  transactionDate: "2026-06-28"
};

const fields: Array<{ key: keyof TransactionFormState; label: string; mono?: boolean; type?: string }> = [
  { key: "id", label: "Transaction ID", mono: true },
  { key: "institutionId", label: "Institution ID", mono: true },
  { key: "fiscalYear", label: "Fiscal year", mono: true },
  { key: "amountMinor", label: "Amount (minor units)", mono: true },
  { key: "currency", label: "Currency", mono: true },
  { key: "transactionDate", label: "Transaction date", type: "date" }
];

export default function ValidationPage() {
  const [form, setForm] = useState(initialForm);
  const [result, setResult] = useState<ApiResult | null>(null);
  const validate = useValidateTransaction();
  const create = useCreateTransaction();

  const busy = validate.isPending || create.isPending;

  async function handleValidate() {
    const outcome = await validate.mutateAsync(toTransactionPayload(form));
    setResult(outcome);
  }

  async function handleCreate() {
    const outcome = await create.mutateAsync(toTransactionPayload(form));
    setResult(outcome);
    if (outcome.ok) {
      toast.success(`Transaction ${form.id} created`);
    } else {
      toast.error(outcome.error);
    }
  }

  return (
    <>
      <PageHeader
        title="Validation workbench"
        description="Review transaction payloads before persistence and public audit publication."
      />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Transaction payload</CardTitle>
            <CardDescription className="font-mono text-xs">POST /v1/transactions</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            {fields.map((field) => (
              <div key={field.key} className="grid gap-1.5">
                <Label htmlFor={`field-${field.key}`}>{field.label}</Label>
                <Input
                  id={`field-${field.key}`}
                  type={field.type}
                  className={field.mono ? "font-mono" : undefined}
                  value={form[field.key]}
                  onChange={(event) => setForm({ ...form, [field.key]: event.target.value })}
                />
              </div>
            ))}
            <div className="grid gap-1.5 sm:col-span-2">
              <Label htmlFor="field-description">Description</Label>
              <Input
                id="field-description"
                value={form.description}
                onChange={(event) => setForm({ ...form, description: event.target.value })}
              />
            </div>
            <div className="flex gap-2 sm:col-span-2">
              <Button onClick={handleValidate} disabled={busy}>
                Validate transaction
              </Button>
              <Button variant="outline" onClick={handleCreate} disabled={busy}>
                Create transaction
              </Button>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              Validation result
              {result ? <StatusBadge status={result.ok ? "VALIDATED" : "REJECTED"} /> : null}
            </CardTitle>
            <CardDescription>
              {result ? `HTTP ${result.status}` : "Submit a payload to validate it against the core API."}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <pre className="overflow-x-auto rounded-lg bg-muted p-4 font-mono text-xs leading-relaxed">
              {result
                ? JSON.stringify(result.ok ? (result.data ?? { status: result.status }) : { error: result.error }, null, 2)
                : JSON.stringify({ status: "ready" }, null, 2)}
            </pre>
          </CardContent>
        </Card>
      </div>
    </>
  );
}
