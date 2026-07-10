import { buildQuery, formatMoney } from "./format";

test("formatMoney renders minor units with currency", () => {
  expect(formatMoney(1250075, "USD")).toBe("USD 12,500.75");
  expect(formatMoney(0, "USD")).toBe("USD 0.00");
  expect(formatMoney(-50000, "EUR")).toBe("EUR -500.00");
});

test("buildQuery keeps only set parameters, in a stable order", () => {
  expect(
    buildQuery({ page: 2, pageSize: 15, institutionId: "", fiscalYear: undefined }),
  ).toBe("?page=2&pageSize=15");
  expect(buildQuery({})).toBe("");
});
