import { act, renderHook } from "@testing-library/react";
import { MemoryRouter, useSearchParams } from "react-router";
import { describe, expect, it } from "vitest";
import { useTableUrlState } from "./use-table-url-state";

function wrapper(initialEntry: string) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <MemoryRouter initialEntries={[initialEntry]}>{children}</MemoryRouter>;
  };
}

describe("useTableUrlState", () => {
  it("defaults to page 1 and pageSize 25 when the URL is empty", () => {
    const { result } = renderHook(() => useTableUrlState(), { wrapper: wrapper("/transactions") });

    expect(result.current.page).toBe(1);
    expect(result.current.pageSize).toBe(25);
    expect(result.current.filters).toEqual({});
  });

  it("reads table state from URL query parameters", () => {
    const { result } = renderHook(() => useTableUrlState(), {
      wrapper: wrapper("/transactions?page=3&pageSize=10&institutionId=minfin&dateFrom=2026-01-01")
    });

    expect(result.current.page).toBe(3);
    expect(result.current.pageSize).toBe(10);
    expect(result.current.filters).toEqual({ institutionId: "minfin", dateFrom: "2026-01-01" });
  });

  it("writes filters to the URL and resets to page 1", () => {
    const { result } = renderHook(
      () => ({ table: useTableUrlState(), search: useSearchParams()[0] }),
      { wrapper: wrapper("/transactions?page=4") }
    );

    act(() => {
      result.current.table.setFilter("institutionId", "health");
    });

    expect(result.current.search.get("institutionId")).toBe("health");
    expect(result.current.search.get("page")).toBeNull(); // page 1 is the default → omitted
    expect(result.current.table.page).toBe(1);
  });

  it("clears a filter when set to undefined", () => {
    const { result } = renderHook(
      () => ({ table: useTableUrlState(), search: useSearchParams()[0] }),
      { wrapper: wrapper("/transactions?institutionId=minfin") }
    );

    act(() => {
      result.current.table.setFilter("institutionId", undefined);
    });

    expect(result.current.search.get("institutionId")).toBeNull();
  });

  it("changes page without disturbing filters", () => {
    const { result } = renderHook(
      () => ({ table: useTableUrlState(), search: useSearchParams()[0] }),
      { wrapper: wrapper("/transactions?institutionId=minfin&pageSize=10") }
    );

    act(() => {
      result.current.table.setPage(2);
    });

    expect(result.current.search.get("page")).toBe("2");
    expect(result.current.search.get("institutionId")).toBe("minfin");
    expect(result.current.search.get("pageSize")).toBe("10");
  });
});
