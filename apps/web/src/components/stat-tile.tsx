import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

type StatTileProps = {
  label: string;
  value: string;
  hint?: string;
  tone?: "up" | "down" | "neutral";
};

export function StatTile({ label, value, hint, tone = "neutral" }: StatTileProps) {
  return (
    <Card>
      <CardContent className="px-4">
        <div className="text-[13px] text-muted-foreground">{label}</div>
        <div className="mt-0.5 text-[26px] font-semibold tracking-tight">{value}</div>
        {hint ? (
          <div
            className={cn(
              "text-xs font-semibold",
              tone === "up" && "text-success",
              tone === "down" && "text-destructive",
              tone === "neutral" && "text-muted-foreground"
            )}
          >
            {hint}
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}
