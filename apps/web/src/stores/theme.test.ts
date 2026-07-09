import { beforeEach, describe, expect, it } from "vitest";
import { applyTheme, useThemeStore } from "./theme";

describe("theme store", () => {
  beforeEach(() => {
    useThemeStore.setState({ theme: "dark" });
    document.documentElement.classList.remove("dark");
  });

  it("defaults to dark (operator dashboards are dark-first)", () => {
    expect(useThemeStore.getState().theme).toBe("dark");
  });

  it("toggles between dark and light", () => {
    useThemeStore.getState().toggle();
    expect(useThemeStore.getState().theme).toBe("light");
    useThemeStore.getState().toggle();
    expect(useThemeStore.getState().theme).toBe("dark");
  });

  it("applies the dark class to the document root", () => {
    applyTheme("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    applyTheme("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("persists the choice under the opentreasury key", () => {
    useThemeStore.getState().setTheme("light");
    expect(window.localStorage.getItem("opentreasury-theme")).toContain("light");
  });
});
