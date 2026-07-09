import { useState } from "react";
import { ListChecks } from "lucide-react";
import { EmptyState } from "@/components/empty-state";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from "@/components/ui/table";
import { useTransactions } from "@/hooks/use-treasury";
import { formatTransactionAmount, toTransactionListFilter, type HistoryFilterForm } from "@/history";

const initialFilters: HistoryFilterForm = { institutionId: "", fiscalYear: "", limit: "25" };

export default function TransactionsPage() {
  const [form, setForm] = useState<HistoryFilterForm>(initialFilters);
  const [applied, setApplied] = useState(() => toTransactionListFilter(initialFilters));
  const transactions = useTransactions(applied);

  const rows = transactions.data?.ok ? transactions.data.transactions : [];

  return (
    <>
      <PageHeader title="Transactions" description="Browse recorded treasury transactions." />
      <Card>
        <CardContent className="flex flex-wrap items-end gap-3">
          <div className="grid gap-1.5">
            <Label htmlFor="filter-institution">Institution</Label>
            <Input
              id="filter-institution"
              placeholder="e.g. minfin"
              className="w-44 font-mono"
              value={form.institutionId}
              onChange={(event) => setForm({ ...form, institutionId: event.target.value })}
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="filter-year">Fiscal year</Label>
            <Input
              id="filter-year"
              placeholder="2026"
              className="w-28 font-mono"
              value={form.fiscalYear}
              onChange={(event) => setForm({ ...form, fiscalYear: event.target.value })}
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="filter-limit">Limit</Label>
            <Input
              id="filter-limit"
              className="w-24 font-mono"
              value={form.limit}
              onChange={(event) => setForm({ ...form, limit: event.target.value })}
            />
          </div>
          <Button onClick={() => setApplied(toTransactionListFilter(form))}>Apply filters</Button>
        </CardContent>
      </Card>
      <Card className="py-0">
        <CardContent className="px-0">
          {transactions.isPending ? (
            <div className="space-y-2 p-4">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
            </div>
          ) : rows.length === 0 ? (
            <EmptyState
              icon={ListChecks}
              message={
                transactions.data?.ok === false
                  ? transactions.data.error
                  : "No transactions match these filters. Widen the fiscal year or clear the institution."
              }
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Transaction</TableHead>
                  <TableHead>Institution</TableHead>
                  <TableHead>Fiscal year</TableHead>
                  <TableHead>Date</TableHead>
                  <TableHead className="text-right">Amount</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((tx) => (
                  <TableRow key={tx.id}>
                    <TableCell className="font-mono text-[13px]">{tx.id}</TableCell>
                    <TableCell>{tx.institutionId}</TableCell>
                    <TableCell className="figures-tabular">{tx.fiscalYear}</TableCell>
                    <TableCell className="figures-tabular">{tx.transactionDate}</TableCell>
                    <TableCell className="text-right figures-tabular">
                      {formatTransactionAmount(tx.currency, tx.amountMinor)}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status="CREATED" />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </>
  );
}
