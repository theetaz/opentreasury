import { useCallback, useMemo } from "react";
import { useSearchParams } from "react-router";

export const DEFAULT_PAGE_SIZE = 25;

const reservedKeys = new Set(["page", "pageSize"]);

export type TableUrlState = {
  page: number;
  pageSize: number;
  /** Every non-pagination query parameter, verbatim. */
  filters: Record<string, string>;
  /** Set (or clear with undefined) a filter; resets to page 1. */
  setFilter: (key: string, value: string | undefined) => void;
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
};

// Table state lives in the URL (mandatory convention): views are shareable,
// bookmarkable, and survive reloads. Defaults are omitted from the URL.
export function useTableUrlState(): TableUrlState {
  const [searchParams, setSearchParams] = useSearchParams();

  const page = positiveInt(searchParams.get("page")) ?? 1;
  const pageSize = positiveInt(searchParams.get("pageSize")) ?? DEFAULT_PAGE_SIZE;

  const filters = useMemo(() => {
    const collected: Record<string, string> = {};
    for (const [key, value] of searchParams.entries()) {
      if (!reservedKeys.has(key) && value !== "") {
        collected[key] = value;
      }
    }
    return collected;
  }, [searchParams]);

  const setFilter = useCallback(
    (key: string, value: string | undefined) => {
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        if (value == null || value === "") {
          next.delete(key);
        } else {
          next.set(key, value);
        }
        next.delete("page"); // filter changes restart at page 1
        return next;
      });
    },
    [setSearchParams]
  );

  const setPage = useCallback(
    (nextPage: number) => {
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        if (nextPage <= 1) {
          next.delete("page");
        } else {
          next.set("page", String(nextPage));
        }
        return next;
      });
    },
    [setSearchParams]
  );

  const setPageSize = useCallback(
    (nextPageSize: number) => {
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        if (nextPageSize === DEFAULT_PAGE_SIZE) {
          next.delete("pageSize");
        } else {
          next.set("pageSize", String(nextPageSize));
        }
        next.delete("page");
        return next;
      });
    },
    [setSearchParams]
  );

  return { page, pageSize, filters, setFilter, setPage, setPageSize };
}

function positiveInt(raw: string | null): number | undefined {
  if (!raw) return undefined;
  const value = Number(raw);
  return Number.isInteger(value) && value > 0 ? value : undefined;
}
