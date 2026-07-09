import { Outlet, useLocation } from "react-router";
import { AppSidebar } from "@/components/app-sidebar";
import { HealthIndicator } from "@/components/health-indicator";
import { ThemeToggle } from "@/components/theme-toggle";
import { UserMenu } from "@/components/user-menu";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";

const titles: Record<string, string> = {
  "/": "Overview",
  "/transactions": "Transactions",
  "/validation": "Validation",
  "/accounts": "Chart of accounts",
  "/journal": "Journal",
  "/balances": "Balances",
  "/ingestion": "Ingestion",
  "/institutions": "Institutions",
  "/audit": "Audit trail"
};

const environment = import.meta.env.VITE_CORE_API_URL ? "LOCAL" : "DEMO";

export function AppShell() {
  const location = useLocation();

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-14 items-center gap-3 border-b px-4">
          <SidebarTrigger aria-label="Toggle sidebar" />
          <Separator orientation="vertical" className="!h-4" />
          <span className="text-sm text-muted-foreground">
            Treasury / <span className="font-semibold text-foreground">{titles[location.pathname] ?? "OpenTreasury"}</span>
          </span>
          <Badge variant="outline" className="border-info/30 bg-info/10 font-bold tracking-wide text-info">
            {environment}
          </Badge>
          <div className="ml-auto flex items-center gap-3">
            <HealthIndicator />
            <ThemeToggle />
            <UserMenu />
          </div>
        </header>
        <main className="flex flex-1 flex-col gap-4 p-4 md:p-6">
          <Outlet />
        </main>
      </SidebarInset>
      <Toaster position="bottom-right" />
    </SidebarProvider>
  );
}
