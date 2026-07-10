import type { ColumnDef } from "@tanstack/react-table";
import { Building2 } from "lucide-react";
import { DataTable } from "@/components/data-table/data-table";
import { FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { useInstitutionsPage } from "@/hooks/use-treasury";
import type { Institution } from "@/api";

const columns: ColumnDef<Institution, unknown>[] = [
  {
    accessorKey: "name",
    header: "Institution",
    cell: ({ row }) => (
      <div>
        <div className="font-medium">{row.original.name}</div>
        <div className="font-mono text-xs text-muted-foreground">{row.original.id}</div>
      </div>
    )
  },
  { accessorKey: "type", header: "Type" },
  { accessorKey: "countryCode", header: "Country" },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => <StatusBadge status={row.original.status} />
  }
];

export default function InstitutionsPage() {
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();
  const institutions = useInstitutionsPage({ status: filters.status, page, pageSize });
  const result = institutions.data;

  return (
    <>
      <PageHeader title="Institutions" description="Government entities reporting into the treasury." />
      <FilterBar>
        <SelectFilter
          label="Status"
          value={filters.status}
          options={[
            { value: "ACTIVE", label: "Active" },
            { value: "INACTIVE", label: "Inactive" }
          ]}
          onChange={(value) => setFilter("status", value)}
        />
      </FilterBar>
      <DataTable
        columns={columns}
        data={result?.ok ? result.institutions : []}
        total={result?.ok ? result.pagination.total : 0}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
        isPending={institutions.isPending}
        error={result?.ok === false ? result.error : undefined}
        emptyMessage="No institutions match this status."
        emptyIcon={Building2}
      />
    </>
  );
}
