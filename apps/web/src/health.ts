import type { HealthCheckResult } from "./api";

export type HealthDisplay = {
  label: string;
  tone: "checking" | "healthy" | "unreachable";
  responseTime: string;
  checkedAt: string;
};

export function toHealthDisplay(result: HealthCheckResult | null): HealthDisplay {
  if (!result) {
    return {
      label: "Checking",
      tone: "checking",
      responseTime: "n/a",
      checkedAt: "pending"
    };
  }

  return {
    label: result.ok ? "Healthy" : "Unreachable",
    tone: result.ok ? "healthy" : "unreachable",
    responseTime: result.ok ? `${result.responseTimeMs} ms` : "n/a",
    checkedAt: formatCheckedAt(result.checkedAt)
  };
}

function formatCheckedAt(value: string): string {
  return new Date(value).toISOString().slice(11, 19);
}
