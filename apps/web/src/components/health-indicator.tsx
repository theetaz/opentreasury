import { cn } from "@/lib/utils";
import { useHealth } from "@/hooks/use-treasury";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

export function HealthIndicator() {
  const { data } = useHealth();

  const healthy = data?.ok === true;
  const label = data == null ? "Checking API…" : healthy ? "API healthy" : "API unreachable";

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="flex items-center gap-2 text-xs text-muted-foreground" aria-live="polite">
          <span
            aria-hidden
            className={cn(
              "size-2 rounded-full",
              data == null && "bg-muted-foreground/50",
              data != null && (healthy ? "bg-success" : "bg-destructive")
            )}
          />
          {label}
        </div>
      </TooltipTrigger>
      <TooltipContent>
        {data?.ok
          ? `${data.responseTimeMs} ms · checked ${new Date(data.checkedAt).toLocaleTimeString()}`
          : data?.error ?? "Waiting for first check"}
      </TooltipContent>
    </Tooltip>
  );
}
