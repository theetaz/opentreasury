import { describe, expect, it } from "vitest";
import {
  buildTransactionsPath,
  formatTransactionAmount,
  toTransactionListFilter,
  toRecentEvent
} from "./history";

describe("buildTransactionsPath", () => {
  it("builds a filtered transaction history path", () => {
    expect(
      buildTransactionsPath({
        institutionId: "minfin",
        fiscalYear: 2026,
        limit: 5
      })
    ).toBe("/v1/transactions?institutionId=minfin&fiscalYear=2026&limit=5");
  });
});

describe("toTransactionListFilter", () => {
  it("normalizes history filter form values", () => {
    expect(
      toTransactionListFilter({
        institutionId: " minfin ",
        fiscalYear: "2026",
        limit: "10"
      })
    ).toEqual({
      institutionId: "minfin",
      fiscalYear: 2026,
      limit: 10
    });
  });

  it("drops empty and invalid filter values", () => {
    expect(
      toTransactionListFilter({
        institutionId: " ",
        fiscalYear: "not-a-year",
        limit: "-5"
      })
    ).toEqual({});
  });

  it("caps limit at the API maximum of 100", () => {
    expect(
      toTransactionListFilter({
        institutionId: "",
        fiscalYear: "",
        limit: "500"
      })
    ).toEqual({ limit: 100 });
  });
});

describe("formatTransactionAmount", () => {
  it("formats minor-unit transaction amounts", () => {
    expect(formatTransactionAmount("USD", 125000)).toBe("USD 1,250.00");
  });
});

describe("toRecentEvent", () => {
  it("maps a treasury transaction into a recent activity row", () => {
    expect(
      toRecentEvent({
        id: "txn-2026-0001",
        institutionId: "minfin",
        fiscalYear: 2026,
        amountMinor: 125000,
        currency: "USD",
        description: "Road maintenance payment",
        transactionDate: "2026-06-28"
      })
    ).toEqual({
      id: "txn-2026-0001",
      institution: "minfin",
      date: "2026-06-28",
      amount: "USD 1,250.00",
      status: "created",
      time: "Recorded"
    });
  });
});
