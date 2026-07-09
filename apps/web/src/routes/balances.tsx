import { useMemo } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Scale } from "lucide-react";
import { DataTable } from "@/components/data-table/data-table";
import { FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { useBalances, useInstitutions } from "@/hooks/use-treasury";
import { formatMoney } from "@/lib/money";
import { cn } from "@/lib/utils";
import type { Balance } from "@/api";

const typeTone: Record<string, string> = {
  REVENUE: "bg-success/12 text-success",
  EXPENSE: "bg-warning/14 text-warning",
  ASSET: "bg-info/12 text-info",
  LIABILITY: "bg-destructive/12 text-destructive",
  NET_WORTH: "bg-muted text-muted-foreground"
};

const columns: ColumnDef<Balance, unknown>[] = [
  {
    accessorKey: "accountCode",
    header: "Account",
    cell: ({ row }) => (
      <div>
        <div className="font-medium">{row.original.accountName || row.original.accountCode}</div>
        <div className="font-mono text-xs text-muted-foreground">{row.original.accountCode}</div>
      </div>
    )
  },
  {
    accessorKey: "accountType",
    header: "Type",
    cell: ({ row }) =>
      row.original.accountType ? (
        <Badge variant="outline" className={cn("border-transparent font-semibold", typeTone[row.original.accountType])}>
          {row.original.accountType.replace("_", " ")}
        </Badge>
      ) : null
  },
  { accessorKey: "institutionId", header: "Institution" },
  { accessorKey: "currency", header: "Currency" },
  {
    accessorKey: "balanceMinor",
    header: () => <div className="text-right">Balance</div>,
    cell: ({ row }) => (
      <div
        className={cn(
          "text-right figures-tabular",
          row.original.balanceMinor < 0 && "text-destructive"
        )}
      >
        {formatMoney(row.original.currency, row.original.balanceMinor)}
      </div>
    )
  }
];

export default function BalancesPage() {
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();
  const institutions = useInstitutions();

  const apiFilter = useMemo(
    () => ({ institutionId: filters.institutionId, page, pageSize }),
    [filters, page, pageSize]
  );

  const balances = useBalances(apiFilter);
  const result = balances.data;

  const institutionOptions = institutions.data?.ok
    ? institutions.data.institutions.map((i) => ({ value: i.id, label: i.name }))
    : [];

  return (
    <>
      <PageHeader
        title="Balances"
        description="Materialized stocks — the net posted position of each account."
      />
      <FilterBar>
        <SelectFilter
          label="Institution"
          value={filters.institutionId}
          options={institutionOptions}
          onChange={(value) => setFilter("institutionId", value)}
        />
      </FilterBar>
      <DataTable
        columns={columns}
        data={result?.ok ? result.balances : []}
        total={result?.ok ? result.pagination.total : 0}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
        isPending={balances.isPending}
        error={result?.ok === false ? result.error : undefined}
        emptyMessage="No balances yet — post a journal entry to move stocks."
        emptyIcon={Scale}
      />
    </>
  );
}
