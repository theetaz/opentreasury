import type { RouteObject } from "react-router";
import { AppShell } from "@/components/app-shell";
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
      { path: "institutions", element: <InstitutionsPage /> },
      { path: "audit", element: <AuditPage /> }
    ]
  }
];
