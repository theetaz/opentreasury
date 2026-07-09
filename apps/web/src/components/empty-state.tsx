import type { LucideIcon } from "lucide-react";

export function EmptyState({
  icon: Icon,
  message,
  action
}: {
  icon: LucideIcon;
  message: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center gap-2 px-4 py-10 text-center text-[13.5px] text-muted-foreground">
      <Icon className="size-5 text-muted-foreground/70" aria-hidden />
      <p>{message}</p>
      {action}
    </div>
  );
}
