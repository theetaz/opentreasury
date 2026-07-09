import type { RouteObject } from "react-router";
import { AppShell } from "@/components/app-shell";
import AccountsPage from "./accounts";
import BalancesPage from "./balances";
import JournalPage from "./journal";
import AuditPage from "./audit";
import InstitutionsPage from "./institutions";
import OverviewPage from "./overview";
import TransactionsPage from "./transactions";
import ValidationPage from "./validation";

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <OverviewPage /> },
      { path: "transactions", element: <TransactionsPage /> },
      { path: "validation", element: <ValidationPage /> },
      { path: "accounts", element: <AccountsPage /> },
      { path: "journal", element: <JournalPage /> },
      { path: "balances", element: <BalancesPage /> },
      { path: "institutions", element: <InstitutionsPage /> },
      { path: "audit", element: <AuditPage /> }
    ]
  }
];
