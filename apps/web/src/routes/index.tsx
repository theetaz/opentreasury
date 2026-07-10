import { lazy, Suspense } from "react";
import type { RouteObject } from "react-router";
import { AppShell } from "@/components/app-shell";

// Route bodies are code-split: heavy dependencies (charts on the overview,
// dialogs on the journal) load with their page instead of on first paint.
const OverviewPage = lazy(() => import("./overview"));
const TransactionsPage = lazy(() => import("./transactions"));
const ValidationPage = lazy(() => import("./validation"));
const AccountsPage = lazy(() => import("./accounts"));
const JournalPage = lazy(() => import("./journal"));
const BalancesPage = lazy(() => import("./balances"));
const IngestionPage = lazy(() => import("./ingestion"));
const InstitutionsPage = lazy(() => import("./institutions"));
const AuditPage = lazy(() => import("./audit"));
const NotFoundPage = lazy(() => import("./not-found"));

function page(element: React.ReactNode) {
  return (
    <Suspense
      fallback={
        <div className="p-6 text-sm text-muted-foreground" role="status">
          Loading…
        </div>
      }
    >
      {element}
    </Suspense>
  );
}

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: page(<OverviewPage />) },
      { path: "transactions", element: page(<TransactionsPage />) },
      { path: "validation", element: page(<ValidationPage />) },
      { path: "accounts", element: page(<AccountsPage />) },
      { path: "journal", element: page(<JournalPage />) },
      { path: "balances", element: page(<BalancesPage />) },
      { path: "ingestion", element: page(<IngestionPage />) },
      { path: "institutions", element: page(<InstitutionsPage />) },
      { path: "audit", element: page(<AuditPage />) },
      { path: "*", element: page(<NotFoundPage />) }
    ]
  }
];
