import { describe, expect, it } from "vitest";
import { buildAuditEventsPath, prependAuditRow, toAuditRow } from "./audit";

describe("buildAuditEventsPath", () => {
  it("builds a filtered audit event path", () => {
    expect(
      buildAuditEventsPath({
        institutionId: "minfin",
        limit: 10
      })
    ).toBe("/v1/audit-events?institutionId=minfin&limit=10");
  });
});

describe("toAuditRow", () => {
  it("maps an audit event into a display row", () => {
    expect(
      toAuditRow({
        id: "audit-txn-2026-0001-created",
        eventType: "TRANSACTION_CREATED",
        transactionId: "txn-2026-0001",
        institutionId: "minfin",
        occurredAt: "2026-06-28T10:24:28Z",
        summary: "Transaction txn-2026-0001 was created."
      })
    ).toEqual({
      id: "audit-txn-2026-0001-created",
      eventType: "Transaction created",
      transactionId: "txn-2026-0001",
      institution: "minfin",
      occurredAt: "2026-06-28 10:24:28",
      summary: "Transaction txn-2026-0001 was created."
    });
  });
});

describe("prependAuditRow", () => {
  it("replaces an existing audit row with the same id", () => {
    const existing = {
      id: "audit-txn-2026-0001-created",
      eventType: "Transaction created",
      transactionId: "txn-2026-0001",
      institution: "minfin",
      occurredAt: "2026-06-28 10:24:28",
      summary: "Old summary"
    };
    const replacement = {
      ...existing,
      occurredAt: "2026-06-28 10:30:00",
      summary: "New summary"
    };

    expect(prependAuditRow([existing], replacement)).toEqual([replacement]);
  });
});
