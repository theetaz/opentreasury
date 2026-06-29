import { describe, expect, it } from "vitest";
import { buildInstitutionsPath, toInstitutionRow } from "./institutions";

describe("buildInstitutionsPath", () => {
  it("builds a limited institutions path", () => {
    expect(buildInstitutionsPath({ limit: 25 })).toBe("/v1/institutions?limit=25");
  });
});

describe("toInstitutionRow", () => {
  it("maps an institution into a display row", () => {
    expect(
      toInstitutionRow({
        id: "minfin",
        name: "Ministry of Finance",
        type: "MINISTRY",
        countryCode: "KE",
        status: "ACTIVE"
      })
    ).toEqual({
      id: "minfin",
      name: "Ministry of Finance",
      type: "Ministry",
      countryCode: "KE",
      status: "Active"
    });
  });
});
