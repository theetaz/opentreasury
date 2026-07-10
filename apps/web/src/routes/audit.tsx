import { useMemo } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { ScrollText } from "lucide-react";
import { DataTable } from "@/components/data-table/data-table";
import { DateRangeFilter, FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { useAuditEvents, useInstitutions } from "@/hooks/use-treasury";
import { toAuditRow, type AuditRow } from "@/audit";

const columns: ColumnDef<AuditRow, unknown>[] = [
  {
    id: "event",
    header: "Event",
    cell: () => <StatusBadge status="CREATED" />
  },
  {
    accessorKey: "transactionId",
    header: "Transaction",
    cell: ({ row }) => <span className="font-mono text-[13px]">{row.original.transactionId}</span>
  },
  { accessorKey: "institution", header: "Institution" },
  {
    accessorKey: "occurredAt",
    header: "Occurred",
    cell: ({ row }) => <span className="figures-tabular">{row.original.occurredAt}</span>
  },
  {
    accessorKey: "summary",
    header: "Summary",
    cell: ({ row }) => <span className="text-muted-foreground">{row.original.summary}</span>
  }
];

export default function AuditPage() {
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();
  const institutions = useInstitutions();

  const apiFilter = useMemo(
    () => ({
      institutionId: filters.institutionId,
      dateFrom: filters.dateFrom,
      dateTo: filters.dateTo,
      page,
      pageSize
    }),
    [filters, page, pageSize]
  );

  const events = useAuditEvents(apiFilter);
  const result = events.data;
  const rows = result?.ok ? result.events.map(toAuditRow) : [];

  const institutionOptions = institutions.data?.ok
    ? institutions.data.institutions.map((institution) => ({
        value: institution.id,
        label: institution.name
      }))
    : [];

  return (
    <>
      <PageHeader title="Audit trail" description="Every recorded change, in order, with its origin." />
      <FilterBar>
        <SelectFilter
          label="Institution"
          value={filters.institutionId}
          options={institutionOptions}
          onChange={(value) => setFilter("institutionId", value)}
        />
        <DateRangeFilter
          label="Occurred"
          dateFrom={filters.dateFrom}
          dateTo={filters.dateTo}
          onChange={(dateFrom, dateTo) => {
            setFilter("dateFrom", dateFrom);
            setFilter("dateTo", dateTo);
          }}
        />
      </FilterBar>
      <DataTable
        columns={columns}
        data={rows}
        total={result?.ok ? result.pagination.total : 0}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
        isPending={events.isPending}
        error={result?.ok === false ? result.error : undefined}
        emptyMessage="No audit events in this window."
        emptyIcon={ScrollText}
      />
    </>
  );
}
