// Money is integers in minor units end to end (ADR-0001); formatting to
// major units happens only at the presentation edge.
export function formatMoney(amountMinor: number, currency: string): string {
  const sign = amountMinor < 0 ? "-" : "";
  const absolute = Math.abs(amountMinor);
  const major = Math.floor(absolute / 100).toLocaleString("en-US");
  const cents = String(absolute % 100).padStart(2, "0");
  return `${currency} ${sign}${major}.${cents}`;
}

/** Serializes set parameters into a query string; empty/undefined are dropped. */
export function buildQuery(
  params: Record<string, string | number | undefined>,
): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") {
      query.append(key, String(value));
    }
  }
  const encoded = query.toString();
  return encoded ? `?${encoded}` : "";
}
