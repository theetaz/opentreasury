import { Building2 } from "lucide-react";
import { EmptyState } from "@/components/empty-state";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from "@/components/ui/table";
import { useInstitutions } from "@/hooks/use-treasury";
import { toInstitutionRow } from "@/institutions";

export default function InstitutionsPage() {
  const institutions = useInstitutions();
  const rows = institutions.data?.ok ? institutions.data.institutions.map(toInstitutionRow) : [];

  return (
    <>
      <PageHeader title="Institutions" description="Government entities reporting into the treasury." />
      <Card className="py-0">
        <CardContent className="px-0">
          {institutions.isPending ? (
            <div className="space-y-2 p-4">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
            </div>
          ) : rows.length === 0 ? (
            <EmptyState
              icon={Building2}
              message={
                institutions.data?.ok === false
                  ? institutions.data.error
                  : "No institutions registered yet."
              }
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Institution</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Country</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>
                      <div className="font-medium">{row.name}</div>
                      <div className="font-mono text-xs text-muted-foreground">{row.id}</div>
                    </TableCell>
                    <TableCell>{row.type}</TableCell>
                    <TableCell>{row.countryCode}</TableCell>
                    <TableCell>
                      <StatusBadge status={row.status} />
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
