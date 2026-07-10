import { Bar, BarChart, CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/page-header";
import { StatTile } from "@/components/stat-tile";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { useBalances, useHealth, useInstitutions, useJournalEntries } from "@/hooks/use-treasury";
import { formatMoney } from "@/lib/money";

const flowsConfig = {
  amount: { label: "Flow volume", color: "var(--chart-1)" }
} satisfies ChartConfig;

const stocksConfig = {
  balance: { label: "Balance", color: "var(--chart-2)" }
} satisfies ChartConfig;

export default function OverviewPage() {
  const health = useHealth();
  const entries = useJournalEntries({ pageSize: 100 });
  const balances = useBalances({ pageSize: 100 });
  const institutions = useInstitutions();

  const entryRows = entries.data?.ok ? entries.data.entries : [];
  const balanceRows = balances.data?.ok ? balances.data.balances : [];

  const currency = balanceRows[0]?.currency ?? entryRows[0]?.lines[0]?.currency ?? "USD";

  // Stocks: cash position = net of ASSET-typed balances.
  const assetNetMinor = balanceRows
    .filter((balance) => balance.accountType === "ASSET")
    .reduce((sum, balance) => sum + balance.balanceMinor, 0);

  // Flows: total debit volume posted, and a per-date series.
  const flowByDate = new Map<string, number>();
  let totalFlowMinor = 0;
  for (const entry of entryRows) {
    const debit = entry.lines
      .filter((line) => line.direction === "DEBIT")
      .reduce((sum, line) => sum + line.amountMinor, 0);
    totalFlowMinor += debit;
    flowByDate.set(entry.effectiveDate, (flowByDate.get(entry.effectiveDate) ?? 0) + debit / 100);
  }
  const flowSeries = [...flowByDate.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([date, amount]) => ({ date, amount }));

  const stockSeries = balanceRows
    .filter((balance) => balance.accountType === "ASSET" || balance.accountType === "LIABILITY")
    .map((balance) => ({
      account: balance.accountCode,
      balance: balance.balanceMinor / 100
    }));

  const { t } = useTranslation();
  const isPending = entries.isPending || balances.isPending;

  return (
    <>
      <PageHeader title={t("overview.title")} description={t("overview.description")} />
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatTile
          label="Cash & financial assets"
          value={balanceRows.length > 0 ? formatMoney(currency, assetNetMinor) : "—"}
          hint="net posted position"
          tone={assetNetMinor >= 0 ? "up" : "down"}
        />
        <StatTile
          label="Posted flow volume"
          value={entryRows.length > 0 ? formatMoney(currency, totalFlowMinor) : "—"}
          hint={`${entries.data?.ok ? entries.data.pagination.total : 0} journal entries`}
        />
        <StatTile
          label="Institutions"
          value={institutions.data?.ok ? String(institutions.data.institutions.length) : "—"}
          hint="reporting"
        />
        <StatTile
          label="API latency"
          value={health.data?.ok ? `${health.data.responseTimeMs} ms` : "—"}
          hint={health.data?.ok ? "healthy" : "unreachable"}
          tone={health.data?.ok ? "up" : "down"}
        />
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Flows over time</CardTitle>
            <CardDescription>Posted journal volume by effective date</CardDescription>
          </CardHeader>
          <CardContent>
            {isPending ? (
              <Skeleton className="h-[220px] w-full" />
            ) : (
              <ChartContainer config={flowsConfig} className="h-[220px] w-full">
                <LineChart data={flowSeries} margin={{ left: 12, right: 12 }}>
                  <CartesianGrid vertical={false} stroke="var(--chart-grid)" />
                  <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} />
                  <YAxis tickLine={false} axisLine={false} width={64} />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Line
                    dataKey="amount"
                    type="monotone"
                    stroke="var(--color-amount)"
                    strokeWidth={2}
                    dot={{ r: 3 }}
                  />
                </LineChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Stocks by account</CardTitle>
            <CardDescription>Net balances on asset and liability accounts</CardDescription>
          </CardHeader>
          <CardContent>
            {isPending ? (
              <Skeleton className="h-[220px] w-full" />
            ) : (
              <ChartContainer config={stocksConfig} className="h-[220px] w-full">
                <BarChart data={stockSeries} margin={{ left: 12, right: 12 }}>
                  <CartesianGrid vertical={false} stroke="var(--chart-grid)" />
                  <XAxis dataKey="account" tickLine={false} axisLine={false} tickMargin={8} />
                  <YAxis tickLine={false} axisLine={false} width={64} />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Bar dataKey="balance" fill="var(--color-balance)" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>
      </div>
    </>
  );
}
