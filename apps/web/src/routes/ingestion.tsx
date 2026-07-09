import { useMemo } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { PlugZap } from "lucide-react";
import { DataTable } from "@/components/data-table/data-table";
import { FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useInstitutions, useStagingRecords } from "@/hooks/use-treasury";
import { formatMoney } from "@/lib/money";
import type { StagingRecord } from "@/api";

function recordAmount(record: StagingRecord): { currency: string; amount: number } {
  const debit = record.lines.find((line) => line.direction === "DEBIT");
  return { currency: debit?.currency ?? "USD", amount: debit?.amountMinor ?? 0 };
}

const columns: ColumnDef<StagingRecord, unknown>[] = [
  {
    accessorKey: "sourceRef",
    header: "Source ref",
    cell: ({ row }) => (
      <div>
        <div className="font-mono text-[13px]">{row.original.sourceRef}</div>
        <div className="text-xs text-muted-foreground">{row.original.sourceSystem}</div>
      </div>
    )
  },
  { accessorKey: "institutionId", header: "Institution" },
  {
    accessorKey: "occurredAt",
    header: "Occurred",
    cell: ({ row }) => <span className="figures-tabular">{row.original.occurredAt}</span>
  },
  {
    id: "amount",
    header: () => <div className="text-right">Amount</div>,
    cell: ({ row }) => {
      const { currency, amount } = recordAmount(row.original);
      return <div className="text-right figures-tabular">{formatMoney(currency, amount)}</div>;
    }
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) =>
      row.original.status === "QUARANTINED" && row.original.reason ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="cursor-help">
              <StatusBadge status={row.original.status} />
            </span>
          </TooltipTrigger>
          <TooltipContent>{row.original.reason}</TooltipContent>
        </Tooltip>
      ) : (
        <StatusBadge status={row.original.status} />
      )
  },
  {
    accessorKey: "entryId",
    header: "Journal entry",
    cell: ({ row }) =>
      row.original.entryId ? (
        <span className="font-mono text-xs text-muted-foreground">{row.original.entryId}</span>
      ) : (
        <span className="text-xs text-muted-foreground/60">—</span>
      )
  }
];

export default function IngestionPage() {
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();
  const institutions = useInstitutions();

  const apiFilter = useMemo(
    () => ({ status: filters.status, institutionId: filters.institutionId, page, pageSize }),
    [filters, page, pageSize]
  );

  const staging = useStagingRecords(apiFilter);
  const result = staging.data;

  const institutionOptions = institutions.data?.ok
    ? institutions.data.institutions.map((i) => ({ value: i.id, label: i.name }))
    : [];

  return (
    <>
      <PageHeader
        title="Ingestion"
        description="Records imported from source systems — posted to the journal or quarantined for review."
      />
      <FilterBar>
        <SelectFilter
          label="Status"
          value={filters.status}
          options={[
            { value: "POSTED", label: "Posted" },
            { value: "QUARANTINED", label: "Quarantined" }
          ]}
          onChange={(value) => setFilter("status", value)}
        />
        <SelectFilter
          label="Institution"
          value={filters.institutionId}
          options={institutionOptions}
          onChange={(value) => setFilter("institutionId", value)}
        />
      </FilterBar>
      <DataTable
        columns={columns}
        data={result?.ok ? result.records : []}
        total={result?.ok ? result.pagination.total : 0}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
        isPending={staging.isPending}
        error={result?.ok === false ? result.error : undefined}
        emptyMessage="No ingested records yet — drop a source-system export into the connector."
        emptyIcon={PlugZap}
      />
    </>
  );
}
