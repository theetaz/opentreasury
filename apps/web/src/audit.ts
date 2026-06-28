import type { AuditEvent, AuditEventListFilter } from "./api";

export type AuditRow = {
  id: string;
  eventType: string;
  transactionId: string;
  institution: string;
  occurredAt: string;
  summary: string;
};

export function buildAuditEventsPath(filter: AuditEventListFilter = {}): string {
  const params = new URLSearchParams();

  if (filter.institutionId) {
    params.set("institutionId", filter.institutionId);
  }

  if (filter.limit) {
    params.set("limit", String(filter.limit));
  }

  const query = params.toString();
  return query ? `/v1/audit-events?${query}` : "/v1/audit-events";
}

export function toAuditRow(event: AuditEvent): AuditRow {
  return {
    id: event.id,
    eventType: formatEventType(event.eventType),
    transactionId: event.transactionId,
    institution: event.institutionId,
    occurredAt: formatAuditTimestamp(event.occurredAt),
    summary: event.summary
  };
}

export function prependAuditRow(rows: AuditRow[], row: AuditRow, limit = 5): AuditRow[] {
  return [row, ...rows.filter((current) => current.id !== row.id)].slice(0, limit);
}

function formatEventType(eventType: string): string {
  if (eventType === "TRANSACTION_CREATED") {
    return "Transaction created";
  }

  return eventType;
}

function formatAuditTimestamp(value: string): string {
  return value.replace("T", " ").replace("Z", "");
}
