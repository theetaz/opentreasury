import { Bar, BarChart, CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
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
import { useAuditEvents, useHealth, useInstitutions, useTransactions } from "@/hooks/use-treasury";
import { formatTransactionAmount } from "@/history";

const flowsConfig = {
  amount: { label: "Amount", color: "var(--chart-1)" }
} satisfies ChartConfig;

const timelineConfig = {
  amount: { label: "Amount", color: "var(--chart-1)" }
} satisfies ChartConfig;

export default function OverviewPage() {
  const health = useHealth();
  const transactions = useTransactions({ limit: 50 });
  const institutions = useInstitutions();
  const auditEvents = useAuditEvents({ limit: 50 });

  const txs = transactions.data?.ok ? transactions.data.transactions : [];
  const totalMinor = txs.reduce((sum, tx) => sum + tx.amountMinor, 0);
  const currency = txs[0]?.currency ?? "USD";

  const byInstitution = Object.entries(
    txs.reduce<Record<string, number>>((acc, tx) => {
      acc[tx.institutionId] = (acc[tx.institutionId] ?? 0) + tx.amountMinor / 100;
      return acc;
    }, {})
  ).map(([institution, amount]) => ({ institution, amount }));

  const timeline = [...txs]
    .sort((a, b) => a.transactionDate.localeCompare(b.transactionDate))
    .map((tx) => ({ date: tx.transactionDate, amount: tx.amountMinor / 100 }));

  return (
    <>
      <PageHeader
        title="Overview"
        description="Whole-of-government stocks and flows at a glance."
      />
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatTile
          label="Recorded volume"
          value={txs.length > 0 ? formatTransactionAmount(currency, totalMinor) : "—"}
          hint={`${txs.length} transactions`}
        />
        <StatTile
          label="Institutions"
          value={institutions.data?.ok ? String(institutions.data.institutions.length) : "—"}
          hint="reporting"
        />
        <StatTile
          label="Audit events"
          value={auditEvents.data?.ok ? String(auditEvents.data.events.length) : "—"}
          hint="recorded"
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
            <CardTitle>Transaction amounts over time</CardTitle>
            <CardDescription>Posted transactions by effective date</CardDescription>
          </CardHeader>
          <CardContent>
            {transactions.isPending ? (
              <Skeleton className="h-[220px] w-full" />
            ) : (
              <ChartContainer config={timelineConfig} className="h-[220px] w-full">
                <LineChart data={timeline} margin={{ left: 12, right: 12 }}>
                  <CartesianGrid vertical={false} stroke="var(--chart-grid)" />
                  <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} />
                  <YAxis tickLine={false} axisLine={false} width={56} />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Line
                    dataKey="amount"
                    type="monotone"
                    stroke="var(--color-amount)"
                    strokeWidth={2}
                    dot={false}
                  />
                </LineChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Flows by institution</CardTitle>
            <CardDescription>Total recorded amount per institution</CardDescription>
          </CardHeader>
          <CardContent>
            {transactions.isPending ? (
              <Skeleton className="h-[220px] w-full" />
            ) : (
              <ChartContainer config={flowsConfig} className="h-[220px] w-full">
                <BarChart data={byInstitution} margin={{ left: 12, right: 12 }}>
                  <CartesianGrid vertical={false} stroke="var(--chart-grid)" />
                  <XAxis dataKey="institution" tickLine={false} axisLine={false} tickMargin={8} />
                  <YAxis tickLine={false} axisLine={false} width={56} />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Bar dataKey="amount" fill="var(--color-amount)" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>
      </div>
    </>
  );
}
