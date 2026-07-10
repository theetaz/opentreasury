import { useTranslation } from "react-i18next";
import { Outlet, useLocation } from "react-router";
import { AppSidebar } from "@/components/app-sidebar";
import { HealthIndicator } from "@/components/health-indicator";
import { LanguageToggle } from "@/components/language-toggle";
import { ThemeToggle } from "@/components/theme-toggle";
import { UserMenu } from "@/components/user-menu";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";

const titleKeys: Record<string, string> = {
  "/": "nav.overview",
  "/transactions": "nav.transactions",
  "/validation": "nav.validation",
  "/accounts": "nav.accounts",
  "/journal": "nav.journal",
  "/balances": "nav.balances",
  "/ingestion": "nav.ingestion",
  "/reconciliation": "nav.reconciliation",
  "/institutions": "nav.institutions",
  "/audit": "nav.audit"
};

const environment = import.meta.env.VITE_CORE_API_URL ? "LOCAL" : "DEMO";

export function AppShell() {
  const location = useLocation();
  const { t } = useTranslation();
  const titleKey = titleKeys[location.pathname];

  return (
    <SidebarProvider>
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-2 focus:top-2 focus:z-50 focus:rounded-md focus:bg-primary focus:px-3 focus:py-2 focus:text-sm focus:text-primary-foreground"
      >
        {t("shell.skipToContent")}
      </a>
      <AppSidebar />
      {/* SidebarInset renders the page's single <main>; the skip link targets it. */}
      <SidebarInset id="main-content">
        <header className="flex h-14 items-center gap-3 border-b px-4">
          <SidebarTrigger aria-label={t("shell.toggleSidebar")} />
          <Separator orientation="vertical" className="!h-4" />
          <span className="text-sm text-muted-foreground">
            {t("shell.breadcrumbRoot")} /{" "}
            <span className="font-semibold text-foreground">
              {titleKey ? t(titleKey) : t("nav.product")}
            </span>
          </span>
          <Badge variant="outline" className="border-info/30 bg-info/10 font-bold tracking-wide text-info">
            {environment}
          </Badge>
          <div className="ml-auto flex items-center gap-3">
            <HealthIndicator />
            <LanguageToggle />
            <ThemeToggle />
            <UserMenu />
          </div>
        </header>
        <div className="flex flex-1 flex-col gap-4 p-4 md:p-6">
          <Outlet />
        </div>
      </SidebarInset>
      <Toaster position="bottom-right" />
    </SidebarProvider>
  );
}
