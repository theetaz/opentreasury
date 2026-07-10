import type { ColumnDef } from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import type { Commitment } from "@/api";
import { DataTable } from "@/components/data-table/data-table";
import { FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";
import { useCommitments, useCreateCommitment, useInstitutions } from "@/hooks/use-treasury";
import { formatMoney } from "@/lib/money";

const fiscalYears = ["2024", "2025", "2026", "2027"];

const columns: ColumnDef<Commitment, unknown>[] = [
  { accessorKey: "committedDate", header: "Committed" },
  {
    accessorKey: "id",
    header: "Commitment",
    cell: ({ row }) => (
      <div className="min-w-0">
        <div className="font-medium">{row.original.id}</div>
        <div className="truncate text-xs text-muted-foreground">{row.original.description}</div>
      </div>
    )
  },
  { accessorKey: "institutionId", header: "Institution" },
  { accessorKey: "accountCode", header: "Account" },
  {
    accessorKey: "amountMinor",
    header: "Committed amount",
    cell: ({ row }) => (
      <span className="figures-tabular">
        {formatMoney(row.original.currency, row.original.amountMinor)}
      </span>
    )
  },
  {
    accessorKey: "settledAmountMinor",
    header: "Settled",
    cell: ({ row }) => {
      const { amountMinor, settledAmountMinor, currency } = row.original;
      const percent = amountMinor > 0 ? Math.round((settledAmountMinor / amountMinor) * 100) : 0;
      return (
        <div className="min-w-32">
          <div className="flex items-center justify-between gap-2 text-xs">
            <span className="figures-tabular">{formatMoney(currency, settledAmountMinor)}</span>
            <span className="text-muted-foreground">{percent}%</span>
          </div>
          <div
            className="mt-1 h-1.5 overflow-hidden rounded-full bg-muted"
            role="progressbar"
            aria-valuenow={percent}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label="Settled share of commitment"
          >
            <div className="h-full bg-primary" style={{ width: `${percent}%` }} />
          </div>
        </div>
      );
    }
  },
  {
    accessorKey: "remainingAmountMinor",
    header: "Remaining",
    cell: ({ row }) => (
      <span className="figures-tabular">
        {formatMoney(row.original.currency, row.original.remainingAmountMinor)}
      </span>
    )
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => <StatusBadge status={row.original.status} />
  }
];

export default function CommitmentsPage() {
  const { t } = useTranslation();
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();
  const institutions = useInstitutions();

  const apiFilter = useMemo(
    () => ({
      institutionId: filters.institutionId,
      fiscalYear: filters.fiscalYear ? Number(filters.fiscalYear) : undefined,
      status: filters.status,
      page,
      pageSize
    }),
    [filters, page, pageSize]
  );

  const commitments = useCommitments(apiFilter);
  const result = commitments.data;

  const institutionOptions = institutions.data?.ok
    ? institutions.data.institutions.map((institution) => ({
        value: institution.id,
        label: institution.name
      }))
    : [];

  return (
    <>
      <PageHeader
        title={t("commitments.title")}
        description={t("commitments.description")}
        actions={<NewCommitmentDialog institutionOptions={institutionOptions} />}
      />
      <FilterBar>
        <SelectFilter
          label={t("commitments.institution")}
          value={filters.institutionId}
          options={institutionOptions}
          onChange={(value) => setFilter("institutionId", value)}
        />
        <SelectFilter
          label={t("commitments.fiscalYear")}
          value={filters.fiscalYear}
          options={fiscalYears.map((year) => ({ value: year, label: year }))}
          onChange={(value) => setFilter("fiscalYear", value)}
        />
        <SelectFilter
          label={t("commitments.status")}
          value={filters.status}
          options={["OPEN", "SETTLED", "CANCELLED"].map((status) => ({ value: status, label: status }))}
          onChange={(value) => setFilter("status", value)}
        />
      </FilterBar>
      <DataTable
        columns={columns}
        data={result?.ok ? result.commitments : []}
        total={result?.ok ? result.pagination.total : 0}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
        isPending={commitments.isPending}
        error={result && !result.ok ? result.error : undefined}
        emptyMessage={t("commitments.empty")}
      />
    </>
  );
}

function NewCommitmentDialog({
  institutionOptions
}: {
  institutionOptions: { value: string; label: string }[];
}) {
  const { t } = useTranslation();
  const create = useCreateCommitment();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    id: "",
    institutionId: "",
    fiscalYear: "2026",
    accountCode: "22",
    description: "",
    amountMajor: "",
    currency: "USD",
    committedDate: new Date().toISOString().slice(0, 10)
  });

  const set = (key: keyof typeof form) => (value: string) =>
    setForm((previous) => ({ ...previous, [key]: value }));

  const submit = async () => {
    const amountMinor = Math.round(Number(form.amountMajor) * 100);
    const result = await create.mutateAsync({
      id: form.id.trim(),
      institutionId: form.institutionId,
      fiscalYear: Number(form.fiscalYear),
      accountCode: form.accountCode.trim(),
      description: form.description.trim(),
      amountMinor,
      currency: form.currency.trim().toUpperCase(),
      committedDate: form.committedDate
    });
    if (result.ok) {
      toast.success(t("commitments.created", { id: form.id.trim() }));
      setOpen(false);
      setForm((previous) => ({ ...previous, id: "", description: "", amountMajor: "" }));
    } else {
      toast.error(result.error);
    }
  };

  const valid =
    form.id.trim() !== "" &&
    form.institutionId !== "" &&
    form.accountCode.trim() !== "" &&
    Number(form.amountMajor) > 0;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>{t("commitments.new")}</Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("commitments.new")}</DialogTitle>
          <DialogDescription>{t("commitments.newDescription")}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-3">
          <div className="grid gap-1.5">
            <Label htmlFor="commitment-id">{t("commitments.fieldId")}</Label>
            <Input
              id="commitment-id"
              value={form.id}
              onChange={(event) => set("id")(event.target.value)}
              placeholder="com-2026-0001"
            />
          </div>
          <div className="grid gap-1.5">
            <Label>{t("commitments.institution")}</Label>
            <Select value={form.institutionId} onValueChange={set("institutionId")}>
              <SelectTrigger>
                <SelectValue placeholder={t("commitments.institution")} />
              </SelectTrigger>
              <SelectContent>
                {institutionOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="grid gap-1.5">
              <Label htmlFor="commitment-account">{t("commitments.account")}</Label>
              <Input
                id="commitment-account"
                value={form.accountCode}
                onChange={(event) => set("accountCode")(event.target.value)}
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="commitment-amount">{t("commitments.amount")}</Label>
              <Input
                id="commitment-amount"
                type="number"
                min="0"
                step="0.01"
                value={form.amountMajor}
                onChange={(event) => set("amountMajor")(event.target.value)}
                placeholder="5000.00"
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="grid gap-1.5">
              <Label>{t("commitments.fiscalYear")}</Label>
              <Select value={form.fiscalYear} onValueChange={set("fiscalYear")}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {fiscalYears.map((year) => (
                    <SelectItem key={year} value={year}>
                      {year}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="commitment-date">{t("commitments.date")}</Label>
              <Input
                id="commitment-date"
                type="date"
                value={form.committedDate}
                onChange={(event) => set("committedDate")(event.target.value)}
              />
            </div>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="commitment-description">{t("commitments.fieldDescription")}</Label>
            <Input
              id="commitment-description"
              value={form.description}
              onChange={(event) => set("description")(event.target.value)}
            />
          </div>
        </div>
        <DialogFooter>
          <Button onClick={() => void submit()} disabled={!valid || create.isPending}>
            {create.isPending ? t("commitments.saving") : t("commitments.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
