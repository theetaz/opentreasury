import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const toneByStatus: Record<string, string> = {
  ACTIVE: "bg-success/12 text-success",
  POSTED: "bg-success/12 text-success",
  CREATED: "bg-success/12 text-success",
  VALIDATED: "bg-info/12 text-info",
  PENDING: "bg-warning/14 text-warning",
  QUARANTINED: "bg-destructive/12 text-destructive",
  MATCHED: "bg-success/12 text-success",
  ATTENTION: "bg-warning/14 text-warning",
  DISCREPANCY: "bg-destructive/12 text-destructive",
  REJECTED: "bg-destructive/12 text-destructive",
  INACTIVE: "bg-muted text-muted-foreground",
  REVERSED: "bg-muted text-muted-foreground"
};

export function StatusBadge({ status, className }: { status: string; className?: string }) {
  const tone = toneByStatus[status.toUpperCase()] ?? "bg-muted text-muted-foreground";

  return (
    <Badge variant="outline" className={cn("gap-1.5 border-transparent font-semibold", tone, className)}>
      <span aria-hidden className="size-1.5 rounded-full bg-current" />
      {status.toUpperCase()}
    </Badge>
  );
}
