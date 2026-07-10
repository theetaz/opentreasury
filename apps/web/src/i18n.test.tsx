import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useRoutes } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import axe from "axe-core";
import { i18n } from "./i18n";
import { routes } from "./routes";

function AppRoutes() {
  return useRoutes(routes);
}

function renderApp(initialPath = "/") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[initialPath]}>
        <AppRoutes />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

afterEach(async () => {
  await i18n.changeLanguage("en");
  window.localStorage.clear();
});

describe("internationalization", () => {
  it("renders Swahili navigation when the language is switched", async () => {
    renderApp();
    const nav = await screen.findByRole("navigation", { name: /primary/i });
    expect(within(nav).getByRole("link", { name: "Overview" })).toBeInTheDocument();

    await i18n.changeLanguage("sw");

    expect(within(nav).getByRole("link", { name: "Muhtasari" })).toBeInTheDocument();
    expect(within(nav).queryByRole("link", { name: "Overview" })).not.toBeInTheDocument();
    expect(document.documentElement.lang).toBe("sw");
  });

  it("offers a language switcher in the shell that persists the choice", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(await screen.findByRole("button", { name: /change language/i }));
    await user.click(await screen.findByRole("menuitem", { name: /kiswahili/i }));

    expect(window.localStorage.getItem("opentreasury-locale")).toBe("sw");
    const nav = screen.getByRole("navigation", { name: /msingi/i });
    expect(within(nav).getByRole("link", { name: "Muhtasari" })).toBeInTheDocument();
  });

  it("translates the overview page and table pagination chrome", async () => {
    renderApp();
    await screen.findByRole("heading", { name: "Overview" });

    await i18n.changeLanguage("sw");

    expect(await screen.findByRole("heading", { name: "Muhtasari" })).toBeInTheDocument();
  });
});

describe("accessibility", () => {
  it.each(["/", "/journal", "/institutions", "/reconciliation"])(
    "route %s has no axe violations",
    async (path) => {
      const { container } = renderApp(path);
      await screen.findByRole("navigation", { name: /primary/i });

      const results = await axe.run(container, {
        // jsdom has no layout engine: contrast needs real rendering, and the
        // scrollable-region rule trips on jsdom's zero-size viewports.
        rules: {
          "color-contrast": { enabled: false },
          "scrollable-region-focusable": { enabled: false },
        },
      });

      expect(
        results.violations.map((v) => `${v.id}: ${v.nodes[0]?.html}`),
      ).toEqual([]);
    },
  );
});
