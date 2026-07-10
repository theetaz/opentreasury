import { describe, expect, it } from "vitest";
import { toTransactionPayload } from "./transaction";

describe("toTransactionPayload", () => {
  it("normalizes form values into the core API transaction payload", () => {
    expect(
      toTransactionPayload({
        id: " txn-2026-0002 ",
        institutionId: " treasury ",
        fiscalYear: "2026",
        amountMinor: "450075",
        currency: "usd",
        transactionDate: "2026-06-29",
        description: " Quarterly grant release "
      })
    ).toEqual({
      id: "txn-2026-0002",
      institutionId: "treasury",
      fiscalYear: 2026,
      amountMinor: 450075,
      currency: "USD",
      transactionDate: "2026-06-29",
      description: "Quarterly grant release"
    });
  });
});
