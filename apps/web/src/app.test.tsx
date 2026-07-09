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
