/** Format minor-unit amounts with an explicit ISO currency code. */
export function formatMoney(currency: string, amountMinor: number): string {
  const sign = amountMinor < 0 ? "-" : "";
  const absolute = Math.abs(amountMinor) / 100;
  return `${sign}${currency} ${absolute.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  })}`;
}
