import { describe, expect, it } from "vitest";
import { toHealthDisplay } from "./health";

describe("toHealthDisplay", () => {
  it("maps a healthy API check into topbar labels", () => {
    expect(
      toHealthDisplay({
        ok: true,
        responseTimeMs: 142,
        checkedAt: "2026-06-28T21:42:30.000Z"
      })
    ).toEqual({
      label: "Healthy",
      tone: "healthy",
      responseTime: "142 ms",
      checkedAt: "21:42:30"
    });
  });

  it("maps an unreachable API check into degraded labels", () => {
    expect(
      toHealthDisplay({
        ok: false,
        error: "Core API is not reachable",
        checkedAt: "2026-06-28T21:42:30.000Z"
      })
    ).toEqual({
      label: "Unreachable",
      tone: "unreachable",
      responseTime: "n/a",
      checkedAt: "21:42:30"
    });
  });
});
