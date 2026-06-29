import { describe, expect, it } from "vitest";
import { listAuditEvents, listInstitutions, validateTransaction } from "./api";

describe("validateTransaction", () => {
  it("returns the core API validation result shape in local simulation", async () => {
    await expect(
      validateTransaction({
        id: "txn-2026-0001",
        institutionId: "minfin",
        fiscalYear: 2026,
        amountMinor: 125000,
        currency: "USD",
        description: "Road maintenance payment",
        transactionDate: "2026-06-28"
      })
    ).resolves.toEqual({
      ok: true,
      status: 200,
      data: {
        status: "VALID",
        code: "VALIDATION_SUCCESS",
        message: "Transaction is valid.",
        transactionId: "txn-2026-0001",
        institutionId: "minfin",
        fiscalYear: 2026,
        amountMinor: 125000,
        currency: "USD",
        transactionDate: "2026-06-28"
      }
    });
  });
});

describe("listAuditEvents", () => {
  it("returns simulated transaction creation events in local mode", async () => {
    await expect(listAuditEvents({ institutionId: "minfin", limit: 1 })).resolves.toEqual({
      ok: true,
      events: [
        {
          id: "audit-txn-2026-0001-created",
          eventType: "TRANSACTION_CREATED",
          transactionId: "txn-2026-0001",
          institutionId: "minfin",
          occurredAt: "2026-06-28T10:24:28Z",
          summary: "Transaction txn-2026-0001 was created."
        }
      ]
    });
  });
});

describe("listInstitutions", () => {
  it("returns simulated institutions in local mode", async () => {
    await expect(listInstitutions({ limit: 1 })).resolves.toEqual({
      ok: true,
      institutions: [
        {
          id: "minfin",
          name: "Ministry of Finance",
          type: "MINISTRY",
          countryCode: "KE",
          status: "ACTIVE"
        }
      ]
    });
  });
});
