import { ScrollText } from "lucide-react";
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
import { useAuditEvents } from "@/hooks/use-treasury";
import { toAuditRow } from "@/audit";

export default function AuditPage() {
  const events = useAuditEvents({ limit: 50 });
  const rows = events.data?.ok ? events.data.events.map(toAuditRow) : [];

  return (
    <>
      <PageHeader
        title="Audit trail"
        description="Every recorded change, in order, with its origin."
      />
      <Card className="py-0">
        <CardContent className="px-0">
          {events.isPending ? (
            <div className="space-y-2 p-4">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
            </div>
          ) : rows.length === 0 ? (
            <EmptyState
              icon={ScrollText}
              message={events.data?.ok === false ? events.data.error : "No audit events recorded yet."}
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Event</TableHead>
                  <TableHead>Transaction</TableHead>
                  <TableHead>Institution</TableHead>
                  <TableHead>Occurred</TableHead>
                  <TableHead>Summary</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>
                      <StatusBadge status="CREATED" />
                    </TableCell>
                    <TableCell className="font-mono text-[13px]">{row.transactionId}</TableCell>
                    <TableCell>{row.institution}</TableCell>
                    <TableCell className="figures-tabular">{row.occurredAt}</TableCell>
                    <TableCell className="text-muted-foreground">{row.summary}</TableCell>
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
