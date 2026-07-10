import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Area, AreaChart, CartesianGrid, Line, XAxis, YAxis } from "recharts";
import type { Anomaly } from "@/api";
import { DataTable } from "@/components/data-table/data-table";
import { FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { useAnomalies, useForecast, useInstitutions } from "@/hooks/use-treasury";
import { formatMoney } from "@/lib/money";

const anomalyColumns: ColumnDef<Anomaly, unknown>[] = [
  { accessorKey: "effectiveDate", header: "Date" },
  { accessorKey: "entryId", header: "Entry" },
  { accessorKey: "institutionId", header: "Institution" },
  { accessorKey: "accountCode", header: "Account" },
  {
    accessorKey: "amountMinor",
    header: "Amount",
    cell: ({ row }) => (
      <span className="font-semibold text-warning figures-tabular">
        {formatMoney("USD", row.original.amountMinor)}
      </span>
    )
  },
  {
    accessorKey: "typicalMinor",
    header: "Typical for account",
    cell: ({ row }) => (
      <span className="figures-tabular">{formatMoney("USD", row.original.typicalMinor)}</span>
    )
  },
  {
    accessorKey: "score",
    header: "Score",
    cell: ({ row }) => (
      <span className="figures-tabular">{row.original.score.toFixed(1)}</span>
    )
  }
];

export default function InsightsPage() {
  const { t } = useTranslation();
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();
  const institutions = useInstitutions();

  const forecast = useForecast(filters.institutionId || undefined, 6);
  const anomalies = useAnomalies(filters.institutionId || undefined, page, pageSize);

  const chartData = useMemo(() => {
    if (!forecast.data?.ok) return [];
    const history = forecast.data.history.map((month) => ({
      period: month.period,
      actual: month.totalMinor / 100,
      projected: null as number | null,
      band: null as [number, number] | null
    }));
    const projection = forecast.data.forecast.map((point) => ({
      period: point.period,
      actual: null as number | null,
      projected: point.projectedMinor / 100,
      band: [point.lowMinor / 100, point.highMinor / 100] as [number, number]
    }));
    // Join the lines visually: the projection starts from the last actual.
    if (history.length > 0) {
      history[history.length - 1].projected = history[history.length - 1].actual;
    }
    return [...history, ...projection];
  }, [forecast.data]);

  const institutionOptions = institutions.data?.ok
    ? institutions.data.institutions.map((institution) => ({
        value: institution.id,
        label: institution.name
      }))
    : [];

  return (
    <>
      <PageHeader title={t("insights.title")} description={t("insights.description")} />
      <FilterBar>
        <SelectFilter
          label={t("insights.institution")}
          value={filters.institutionId}
          options={institutionOptions}
          onChange={(value) => setFilter("institutionId", value)}
        />
      </FilterBar>

      <Card>
        <CardHeader>
          <CardTitle>{t("insights.forecastTitle")}</CardTitle>
          <CardDescription>
            {forecast.data?.ok ? forecast.data.method : t("insights.forecastDescription")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {forecast.isPending ? (
            <Skeleton className="h-64 w-full" />
          ) : chartData.length === 0 ? (
            <p className="py-10 text-center text-sm text-muted-foreground">
              {t("insights.noHistory")}
            </p>
          ) : (
            <ChartContainer
              className="h-64 w-full"
              config={{
                actual: { label: t("insights.actual"), color: "var(--chart-1)" },
                projected: { label: t("insights.projected"), color: "var(--chart-2)" }
              }}
            >
              <AreaChart data={chartData} margin={{ left: 12, right: 12 }}>
                <CartesianGrid vertical={false} stroke="var(--chart-grid)" />
                <XAxis dataKey="period" tickLine={false} axisLine={false} />
                <YAxis tickLine={false} axisLine={false} width={80} />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Area
                  dataKey="band"
                  stroke="none"
                  fill="var(--chart-2)"
                  fillOpacity={0.15}
                  connectNulls={false}
                  isAnimationActive={false}
                />
                <Line
                  dataKey="actual"
                  stroke="var(--chart-1)"
                  strokeWidth={2}
                  dot={false}
                  connectNulls={false}
                  isAnimationActive={false}
                />
                <Line
                  dataKey="projected"
                  stroke="var(--chart-2)"
                  strokeWidth={2}
                  strokeDasharray="6 4"
                  dot={false}
                  connectNulls={false}
                  isAnimationActive={false}
                />
              </AreaChart>
            </ChartContainer>
          )}
        </CardContent>
      </Card>

      <div>
        <h2 className="mb-1 text-lg font-semibold">{t("insights.anomaliesTitle")}</h2>
        <p className="mb-3 text-sm text-muted-foreground">
          {anomalies.data?.ok ? anomalies.data.method : t("insights.anomaliesDescription")}
        </p>
        <DataTable
          columns={anomalyColumns}
          data={anomalies.data?.ok ? anomalies.data.anomalies : []}
          total={anomalies.data?.ok ? anomalies.data.pagination.total : 0}
          page={page}
          pageSize={pageSize}
          onPageChange={setPage}
          onPageSizeChange={setPageSize}
          isPending={anomalies.isPending}
          error={anomalies.data && !anomalies.data.ok ? anomalies.data.error : undefined}
          emptyMessage={t("insights.noAnomalies")}
        />
      </div>
    </>
  );
}
