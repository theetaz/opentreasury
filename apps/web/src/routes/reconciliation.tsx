import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { ReconciliationRow } from "@/api";
import { DataTable } from "@/components/data-table/data-table";
import { FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { useReconciliation } from "@/hooks/use-treasury";
import { formatMoney } from "@/lib/money";

const fiscalYears = ["2024", "2025", "2026", "2027"];

const columns: ColumnDef<ReconciliationRow, unknown>[] = [
  { accessorKey: "period", header: "Period" },
  { accessorKey: "sourceSystem", header: "Source system" },
  {
    accessorKey: "stagedCount",
    header: "Staged",
    cell: ({ row }) => <span className="figures-tabular">{row.original.stagedCount}</span>
  },
  {
    accessorKey: "postedCount",
    header: "Posted",
    cell: ({ row }) => <span className="figures-tabular">{row.original.postedCount}</span>
  },
  {
    accessorKey: "quarantinedCount",
    header: "Quarantined",
    cell: ({ row }) => (
      <span className={row.original.quarantinedCount > 0 ? "font-semibold text-warning" : "figures-tabular"}>
        {row.original.quarantinedCount}
      </span>
    )
  },
  {
    accessorKey: "stagedAmountMinor",
    header: "Staged amount",
    cell: ({ row }) => (
      <span className="figures-tabular">{formatMoney("USD", row.original.stagedAmountMinor)}</span>
    )
  },
  {
    accessorKey: "postedAmountMinor",
    header: "Posted amount",
    cell: ({ row }) => (
      <span className="figures-tabular">{formatMoney("USD", row.original.postedAmountMinor)}</span>
    )
  },
  {
    accessorKey: "discrepancyMinor",
    header: "Discrepancy",
    cell: ({ row }) =>
      row.original.discrepancyMinor === 0 ? (
        <span className="text-muted-foreground">—</span>
      ) : (
        <span className="font-semibold text-danger figures-tabular">
          {formatMoney("USD", row.original.discrepancyMinor)}
        </span>
      )
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => <StatusBadge status={row.original.status} />
  }
];

export default function ReconciliationPage() {
  const { t } = useTranslation();
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();

  const apiFilter = useMemo(
    () => ({
      sourceSystem: filters.sourceSystem,
      fiscalYear: filters.fiscalYear ? Number(filters.fiscalYear) : undefined,
      page,
      pageSize
    }),
    [filters, page, pageSize]
  );

  const reconciliation = useReconciliation(apiFilter);
  const result = reconciliation.data;

  // Source systems present in the current result feed the dropdown; the set
  // is tiny (one per connected government system).
  const sourceOptions = useMemo(() => {
    const sources = new Set<string>(filters.sourceSystem ? [filters.sourceSystem] : []);
    if (result?.ok) {
      for (const row of result.rows) sources.add(row.sourceSystem);
    }
    return [...sources].sort().map((source) => ({ value: source, label: source }));
  }, [result, filters.sourceSystem]);

  return (
    <>
      <PageHeader
        title={t("reconciliation.title")}
        description={t("reconciliation.description")}
      />
      <FilterBar>
        <SelectFilter
          label={t("reconciliation.sourceSystem")}
          value={filters.sourceSystem}
          options={sourceOptions}
          onChange={(value) => setFilter("sourceSystem", value)}
        />
        <SelectFilter
          label={t("reconciliation.fiscalYear")}
          value={filters.fiscalYear}
          options={fiscalYears.map((year) => ({ value: year, label: year }))}
          onChange={(value) => setFilter("fiscalYear", value)}
        />
      </FilterBar>
      <DataTable
        columns={columns}
        data={result?.ok ? result.rows : []}
        total={result?.ok ? result.pagination.total : 0}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
        isPending={reconciliation.isPending}
        error={result && !result.ok ? result.error : undefined}
        emptyMessage={t("reconciliation.empty")}
      />
    </>
  );
}
