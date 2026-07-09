import { useMemo } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { BookOpenCheck } from "lucide-react";
import { DataTable } from "@/components/data-table/data-table";
import { FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { useAccounts } from "@/hooks/use-treasury";
import { cn } from "@/lib/utils";
import type { Account } from "@/api";

const accountTypes = [
  { value: "REVENUE", label: "Revenue" },
  { value: "EXPENSE", label: "Expense" },
  { value: "ASSET", label: "Asset" },
  { value: "LIABILITY", label: "Liability" },
  { value: "NET_WORTH", label: "Net worth" }
];

const typeTone: Record<Account["accountType"], string> = {
  REVENUE: "bg-success/12 text-success",
  EXPENSE: "bg-warning/14 text-warning",
  ASSET: "bg-info/12 text-info",
  LIABILITY: "bg-destructive/12 text-destructive",
  NET_WORTH: "bg-muted text-muted-foreground"
};

const columns: ColumnDef<Account, unknown>[] = [
  {
    accessorKey: "code",
    header: "Code",
    cell: ({ row }) => (
      <span
        className="font-mono text-[13px]"
        style={{ paddingLeft: `${Math.max(0, row.original.code.length - 1) * 12}px` }}
      >
        {row.original.code}
      </span>
    )
  },
  { accessorKey: "name", header: "Account" },
  {
    accessorKey: "accountType",
    header: "Type",
    cell: ({ row }) => (
      <Badge
        variant="outline"
        className={cn("border-transparent font-semibold", typeTone[row.original.accountType])}
      >
        {row.original.accountType.replace("_", " ")}
      </Badge>
    )
  },
  {
    accessorKey: "gfsmCode",
    header: "GFSM 2014",
    cell: ({ row }) =>
      row.original.gfsmCode ? (
        <span className="font-mono text-xs text-muted-foreground">{row.original.gfsmCode}</span>
      ) : null
  },
  {
    accessorKey: "cofogCode",
    header: "COFOG",
    cell: ({ row }) =>
      row.original.cofogCode ? (
        <span className="font-mono text-xs text-muted-foreground">{row.original.cofogCode}</span>
      ) : (
        <span className="text-xs text-muted-foreground/60">—</span>
      )
  }
];

export default function AccountsPage() {
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();

  const apiFilter = useMemo(
    () => ({ accountType: filters.accountType, page, pageSize }),
    [filters, page, pageSize]
  );

  const accounts = useAccounts(apiFilter);
  const result = accounts.data;

  return (
    <>
      <PageHeader
        title="Chart of accounts"
        description="The active account classification — GFSM 2014 aligned, ordered by code."
      />
      <FilterBar>
        <SelectFilter
          label="Account type"
          value={filters.accountType}
          options={accountTypes}
          onChange={(value) => setFilter("accountType", value)}
        />
      </FilterBar>
      <DataTable
        columns={columns}
        data={result?.ok ? result.accounts : []}
        total={result?.ok ? result.pagination.total : 0}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
        isPending={accounts.isPending}
        error={result?.ok === false ? result.error : undefined}
        emptyMessage="No accounts of this type in the active chart."
        emptyIcon={BookOpenCheck}
      />
    </>
  );
}
