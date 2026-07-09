import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useRoutes } from "react-router";
import { describe, expect, it } from "vitest";
import { routes } from "./routes";

// Declarative router in tests: jsdom's AbortSignal is rejected by the data
// router's internal fetch Request, and these tests exercise UI, not loaders.
function AppRoutes() {
  return useRoutes(routes);
}

function renderApp(initialPath = "/") {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } }
  });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[initialPath]}>
        <AppRoutes />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe("app shell", () => {
  it("renders the sidebar navigation", async () => {
    renderApp();

    const nav = await screen.findByRole("navigation", { name: /primary/i });
    for (const item of ["Overview", "Transactions", "Validation", "Institutions", "Audit trail"]) {
      expect(within(nav).getByRole("link", { name: item })).toBeInTheDocument();
    }
  });

  it("navigates between routed pages", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(await screen.findByRole("link", { name: "Institutions" }));
    expect(await screen.findByRole("heading", { name: /institutions/i })).toBeInTheDocument();
    // Simulated data mode (no VITE_CORE_API_URL) serves the demo institutions.
    expect(await screen.findByText("Ministry of Finance")).toBeInTheDocument();
  });
});

describe("unknown routes", () => {
  it("renders the designed not-found page instead of a router error", async () => {
    renderApp("/auth/callback");

    expect(await screen.findByRole("heading", { name: /page not found/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /back to overview/i })).toBeInTheDocument();
  });
});

describe("journal and balances", () => {
  it("lists journal entries with their line counts", async () => {
    renderApp("/journal");

    expect(await screen.findByRole("heading", { name: /^journal$/i })).toBeInTheDocument();
    expect(await screen.findByText("je-2026-0001")).toBeInTheDocument();
    expect(await screen.findByText("Tax receipt into the treasury account")).toBeInTheDocument();
  });

  it("shows materialized balances with signed amounts", async () => {
    renderApp("/balances");

    expect(await screen.findByRole("heading", { name: /balances/i })).toBeInTheDocument();
    expect(await screen.findByText("Currency and deposits")).toBeInTheDocument();
    // The cash account carries a positive net; the revenue account is negative.
    expect(await screen.findByText("USD 1,210.00")).toBeInTheDocument();
  });

  it("blocks posting an unbalanced entry and enables it once balanced", async () => {
    const user = userEvent.setup();
    renderApp("/journal");

    await user.click(await screen.findByRole("button", { name: /post entry/i }));

    const dialog = await screen.findByRole("dialog");
    const post = within(dialog).getByRole("button", { name: /post entry/i });
    expect(post).toBeDisabled();

    await user.type(within(dialog).getByLabelText("Entry ID"), "je-test-1");
    await user.type(within(dialog).getByLabelText("Line 1 amount"), "100");
    await user.type(within(dialog).getByLabelText("Line 2 amount"), "60");
    // Debits (100) != credits (60): still not balanced.
    expect(within(dialog).getByText(/not balanced/i)).toBeInTheDocument();

    await user.clear(within(dialog).getByLabelText("Line 2 amount"));
    await user.type(within(dialog).getByLabelText("Line 2 amount"), "100");
    expect(within(dialog).getByText(/^balanced$/i)).toBeInTheDocument();
  });
});

describe("ingestion", () => {
  it("lists imported records and surfaces quarantine reasons", async () => {
    renderApp("/ingestion?status=QUARANTINED");

    expect(await screen.findByRole("heading", { name: /ingestion/i })).toBeInTheDocument();
    expect(await screen.findByText("PAY-2026-1005")).toBeInTheDocument();
    expect(screen.queryByText("PAY-2026-1001")).not.toBeInTheDocument();
  });
});

describe("chart of accounts", () => {
  it("renders the reference accounts and filters by type from the URL", async () => {
    renderApp("/accounts?accountType=LIABILITY");

    expect(await screen.findByRole("heading", { name: /chart of accounts/i })).toBeInTheDocument();
    expect(await screen.findByText("Liabilities")).toBeInTheDocument();
    expect(screen.queryByText("Compensation of employees")).not.toBeInTheDocument();
  });
});

describe("transactions data table", () => {
  it("reads table state from the URL and renders the matching server page", async () => {
    renderApp("/transactions?institutionId=minfin&pageSize=10");

    // Simulated mode mirrors the server: minfin has exactly one transaction.
    expect(await screen.findByText("txn-2026-0001")).toBeInTheDocument();
    expect(screen.queryByText("txn-2026-0000")).not.toBeInTheDocument();
    expect(await screen.findByText(/1–1 of 1/)).toBeInTheDocument();
  });

  it("filters through the institution dropdown and updates the table", async () => {
    const user = userEvent.setup();
    renderApp("/transactions");

    expect(await screen.findByText("txn-2026-0000")).toBeInTheDocument();

    await user.click(screen.getByRole("combobox", { name: "Institution" }));
    await user.click(await screen.findByRole("option", { name: "Ministry of Health" }));

    expect(await screen.findByText("txn-2025-0942")).toBeInTheDocument();
    expect(screen.queryByText("txn-2026-0001")).not.toBeInTheDocument();
  });

  it("paginates server-side through the footer controls", async () => {
    const user = userEvent.setup();
    renderApp("/transactions");

    expect(await screen.findByText(/1–3 of 3/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next page" })).toBeDisabled();

    await user.click(screen.getByRole("combobox", { name: "Rows per page" }));
    await user.click(await screen.findByRole("option", { name: "25 / page" }));
  });
});

describe("validation workbench", () => {
  it("validates a transaction payload end-to-end (simulated mode)", async () => {
    const user = userEvent.setup();
    renderApp("/validation");

    expect(await screen.findByRole("heading", { name: /validation/i })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /validate transaction/i }));
    expect(await screen.findByText(/VALIDATION_SUCCESS/)).toBeInTheDocument();
  });

  it("shows a rejection when the payload is invalid", async () => {
    const user = userEvent.setup();
    renderApp("/validation");

    const amount = await screen.findByLabelText(/amount/i);
    await user.clear(amount);
    await user.type(amount, "-500");
    await user.click(screen.getByRole("button", { name: /validate transaction/i }));

    expect(await screen.findByText(/invalid amount/i)).toBeInTheDocument();
  });
});
