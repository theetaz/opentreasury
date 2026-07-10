import { useMemo } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { ListChecks } from "lucide-react";
import { DataTable } from "@/components/data-table/data-table";
import { AmountFilter, DateRangeFilter, FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { useInstitutions, useTransactions } from "@/hooks/use-treasury";
import { formatTransactionAmount } from "@/history";
import type { TreasuryTransaction } from "@/api";

const currentYear = new Date().getFullYear();
const fiscalYears = Array.from({ length: 8 }, (_, index) => String(currentYear + 1 - index));

const columns: ColumnDef<TreasuryTransaction, unknown>[] = [
  {
    accessorKey: "id",
    header: "Transaction",
    cell: ({ row }) => <span className="font-mono text-[13px]">{row.original.id}</span>
  },
  { accessorKey: "institutionId", header: "Institution" },
  {
    accessorKey: "fiscalYear",
    header: "Fiscal year",
    cell: ({ row }) => <span className="figures-tabular">{row.original.fiscalYear}</span>
  },
  {
    accessorKey: "transactionDate",
    header: "Date",
    cell: ({ row }) => <span className="figures-tabular">{row.original.transactionDate}</span>
  },
  {
    accessorKey: "amountMinor",
    header: () => <div className="text-right">Amount</div>,
    cell: ({ row }) => (
      <div className="text-right figures-tabular">
        {formatTransactionAmount(row.original.currency, row.original.amountMinor)}
      </div>
    )
  },
  {
    id: "status",
    header: "Status",
    cell: ({ row }) => <StatusBadge status={row.original.status} />
  }
];

export default function TransactionsPage() {
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();
  const institutions = useInstitutions();

  const apiFilter = useMemo(
    () => ({
      institutionId: filters.institutionId,
      fiscalYear: filters.fiscalYear ? Number(filters.fiscalYear) : undefined,
      status: filters.status,
      dateFrom: filters.dateFrom,
      dateTo: filters.dateTo,
      amountGte: filters.amountGte ? Number(filters.amountGte) : undefined,
      amountLte: filters.amountLte ? Number(filters.amountLte) : undefined,
      page,
      pageSize
    }),
    [filters, page, pageSize]
  );

  const transactions = useTransactions(apiFilter);
  const result = transactions.data;

  const institutionOptions = institutions.data?.ok
    ? institutions.data.institutions.map((institution) => ({
        value: institution.id,
        label: institution.name
      }))
    : [];

  return (
    <>
      <PageHeader title="Transactions" description="Browse recorded treasury transactions." />
      <FilterBar>
        <SelectFilter
          label="Institution"
          value={filters.institutionId}
          options={institutionOptions}
          onChange={(value) => setFilter("institutionId", value)}
        />
        <SelectFilter
          label="Fiscal year"
          value={filters.fiscalYear}
          options={fiscalYears.map((year) => ({ value: year, label: year }))}
          onChange={(value) => setFilter("fiscalYear", value)}
        />
        <SelectFilter
          label="Status"
          value={filters.status}
          options={["PENDING", "POSTED", "REJECTED"].map((status) => ({ value: status, label: status }))}
          onChange={(value) => setFilter("status", value)}
        />
        <DateRangeFilter
          dateFrom={filters.dateFrom}
          dateTo={filters.dateTo}
          onChange={(dateFrom, dateTo) => {
            setFilter("dateFrom", dateFrom);
            setFilter("dateTo", dateTo);
          }}
        />
        <AmountFilter
          amountGte={filters.amountGte}
          amountLte={filters.amountLte}
          onChange={(amountGte, amountLte) => {
            setFilter("amountGte", amountGte);
            setFilter("amountLte", amountLte);
          }}
        />
      </FilterBar>
      <DataTable
        columns={columns}
        data={result?.ok ? result.transactions : []}
        total={result?.ok ? result.pagination.total : 0}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
        isPending={transactions.isPending}
        error={result?.ok === false ? result.error : undefined}
        emptyMessage="No transactions match these filters. Widen the date range or clear a filter."
        emptyIcon={ListChecks}
      />
    </>
  );
}
