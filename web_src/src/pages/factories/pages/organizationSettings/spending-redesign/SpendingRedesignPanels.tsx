import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { ChevronDown, CircleDollarSign } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { useElementWidth } from "@/hooks/useElementWidth";
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { SegmentedNav } from "@/ui/SegmentedNav";

import { pickVelocityAxisTicks } from "../../../lib/velocityAxisTicks";
import { formatCompactTokens, formatDurationSeconds, formatUsdCents } from "../../../lib/workOrderUsage";
import { factoryCardClassName } from "../../factoryPageLayoutStyles";
import type { SpendingCreditSnapshot } from "./spendingRedesignMocks";
import {
  EMPTY_SPENDING_FILTERS,
  formatFilterTriggerLabel,
  formatShare,
  SPENDING_BREAKDOWN_OPTIONS,
  toggleFilterValue,
  type SpendingBreakdown,
  type SpendingCatalogItem,
  type SpendingCatalogs,
  type SpendingFilters,
  type SpendingReport,
} from "./spendingRedesignLib";

const SERIES_COLORS = ["#6366f1", "#10b981", "#f59e0b", "#3b82f6", "#8b5cf6", "#94a3b8"];

export function SpendingFilterBar({
  catalogs,
  filters,
  filtersActive,
  onChange,
}: {
  catalogs: SpendingCatalogs;
  filters: SpendingFilters;
  filtersActive: boolean;
  onChange: (filters: SpendingFilters) => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2" data-testid="spending-filter-bar">
      <SpendingFilterMenu
        allLabel="All users"
        noun="user"
        items={catalogs.users}
        selected={filters.userIds}
        testId="spending-filter-users"
        onToggle={(id) => onChange({ ...filters, userIds: toggleFilterValue(filters.userIds, id) })}
      />
      <SpendingFilterMenu
        allLabel="All workspaces"
        noun="workspace"
        items={catalogs.workspaces}
        selected={filters.workspaceIds}
        testId="spending-filter-workspaces"
        onToggle={(id) => onChange({ ...filters, workspaceIds: toggleFilterValue(filters.workspaceIds, id) })}
      />
      <SpendingFilterMenu
        allLabel="All models"
        noun="model"
        items={catalogs.models}
        selected={filters.models}
        testId="spending-filter-models"
        onToggle={(id) => onChange({ ...filters, models: toggleFilterValue(filters.models, id) })}
      />
      <SpendingFilterMenu
        allLabel="All machines"
        noun="machine"
        items={catalogs.machines}
        selected={filters.machineTypes}
        testId="spending-filter-machines"
        onToggle={(id) => onChange({ ...filters, machineTypes: toggleFilterValue(filters.machineTypes, id) })}
      />
      {filtersActive ? (
        <Button
          variant="ghost"
          size="sm"
          data-testid="spending-clear-filters"
          onClick={() => onChange(EMPTY_SPENDING_FILTERS)}
        >
          Clear filters
        </Button>
      ) : null}
    </div>
  );
}

function SpendingFilterMenu({
  allLabel,
  noun,
  items,
  selected,
  testId,
  onToggle,
}: {
  allLabel: string;
  noun: string;
  items: SpendingCatalogItem[];
  selected: string[];
  testId: string;
  onToggle: (id: string) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" data-testid={testId}>
          {formatFilterTriggerLabel(allLabel, selected.length, noun)}
          <ChevronDown className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-48">
        {items.map((item) => (
          <DropdownMenuCheckboxItem
            key={item.id}
            checked={selected.includes(item.id)}
            onCheckedChange={() => onToggle(item.id)}
          >
            {item.label}
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function SpendingKpiRow({
  metrics,
  credit,
  rangeCaption,
}: {
  metrics: { spend: string; tokens: string; duration: string; hosted: string; byok: string };
  credit: SpendingCreditSnapshot;
  rangeCaption: string;
}) {
  return (
    <div className={`grid gap-3 sm:grid-cols-2 xl:grid-cols-4 ${factoryCardClassName} p-4`}>
      <SpendingKpi label="Estimated spend" value={metrics.spend} hint={rangeCaption} testId="spending-kpi-spend" />
      <SpendingKpi
        label="Tokens"
        value={metrics.tokens}
        hint={`Hosted ${metrics.hosted} · Your keys ${metrics.byok}`}
        testId="spending-kpi-tokens"
      />
      <SpendingKpi label="VM time" value={metrics.duration} hint="SuperPlane runner fleets" testId="spending-kpi-vm" />
      <SpendingKpi
        label="Remaining hosted credit"
        value={formatUsdCents(credit.remainingCreditCents)}
        hint="Organization wallet"
        testId="spending-kpi-credit"
      />
    </div>
  );
}

function SpendingKpi({ label, value, hint, testId }: { label: string; value: string; hint: string; testId: string }) {
  return (
    <div data-testid={testId}>
      <p className="workspace-section-label">{label}</p>
      <p className="workspace-page-title mt-1">{value}</p>
      <p className="mt-1 text-[12px] text-muted-foreground">{hint}</p>
    </div>
  );
}

export function SpendingChartCard({ report, breakdown }: { report: SpendingReport; breakdown: SpendingBreakdown }) {
  const empty = report.totals.costCents === 0;
  return (
    <section className={factoryCardClassName}>
      <div className="flex items-start justify-between gap-3 px-4 pt-4">
        <div>
          <h2 className="workspace-section-title">Spend over time</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">
            Stacked by {breakdownLabel(breakdown).toLowerCase()}. SuperPlane-hosted spend and your keys both show.
          </p>
        </div>
      </div>
      {empty ? (
        <SpendingEmptyState />
      ) : (
        <div className="px-2 pb-4 pt-2">
          <SpendingBarChart report={report} />
        </div>
      )}
    </section>
  );
}

function SpendingBarChart({ report }: { report: SpendingReport }) {
  const { ref, width } = useElementWidth<HTMLDivElement>(760);
  const ticks = pickVelocityAxisTicks(
    report.series.map((point) => point.label),
    width,
  );
  const config = Object.fromEntries(
    report.seriesKeys.map((item, index) => [
      item.id,
      { label: item.label, color: SERIES_COLORS[index % SERIES_COLORS.length] },
    ]),
  ) satisfies ChartConfig;
  const rows = report.series.map((point) => ({
    label: point.label,
    ...point.values,
  }));

  return (
    <div ref={ref} className="w-full">
      <ChartContainer
        config={config}
        className="aspect-auto h-[240px] w-full"
        initialDimension={{ width: 760, height: 240 }}
      >
        <BarChart data={rows} margin={{ top: 8, right: 4, left: 8, bottom: 0 }}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
          <XAxis
            dataKey="label"
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            interval={0}
            ticks={ticks}
            className="text-[11px]"
          />
          <YAxis
            tickLine={false}
            axisLine={false}
            width={48}
            className="text-[11px]"
            tickFormatter={(value: number) => formatUsdCents(Number(value))}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                formatter={(value, name) => (
                  <div className="flex w-full items-center justify-between gap-8">
                    <span className="text-muted-foreground">{String(name)}</span>
                    <span className="font-mono font-medium text-foreground tabular-nums">
                      {formatUsdCents(Number(value))}
                    </span>
                  </div>
                )}
              />
            }
          />
          <ChartLegend content={<ChartLegendContent />} verticalAlign="bottom" />
          {report.seriesKeys.map((item, index) => (
            <Bar
              key={item.id}
              dataKey={item.id}
              name={item.label}
              stackId="spend"
              fill={SERIES_COLORS[index % SERIES_COLORS.length]}
              radius={index === report.seriesKeys.length - 1 ? [3, 3, 0, 0] : undefined}
              maxBarSize={28}
            />
          ))}
        </BarChart>
      </ChartContainer>
    </div>
  );
}

export function SpendingBreakdownCard({
  breakdown,
  onBreakdownChange,
  report,
}: {
  breakdown: SpendingBreakdown;
  onBreakdownChange: (value: SpendingBreakdown) => void;
  report: SpendingReport;
}) {
  return (
    <section className={factoryCardClassName} data-testid="spending-breakdown">
      <div className="flex flex-wrap items-center justify-between gap-3 px-4 pt-4">
        <div>
          <h2 className="workspace-section-title">Breakdown</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">
            {breakdown === "user"
              ? "Spend is grouped by the task owner."
              : `One row per ${breakdownLabel(breakdown).toLowerCase().replace(/s$/, "")} in this range.`}
          </p>
        </div>
        <SegmentedNav
          ariaLabel="Spending breakdown"
          options={SPENDING_BREAKDOWN_OPTIONS}
          size="xs"
          value={breakdown}
          onValueChange={(value) => onBreakdownChange(value as SpendingBreakdown)}
        />
      </div>
      {report.breakdown.length === 0 ? (
        <SpendingEmptyState />
      ) : (
        <table className="mt-2 w-full text-left text-[13px]">
          <thead>
            <tr className="border-b border-border text-muted-foreground">
              <th className="px-4 py-2 font-medium">{breakdownColumnLabel(breakdown)}</th>
              <th className="px-4 py-2 font-medium">{breakdown === "machine" ? "Time" : "Tokens"}</th>
              <th className="px-4 py-2 font-medium">Spend</th>
              <th className="px-4 py-2 font-medium">Share</th>
            </tr>
          </thead>
          <tbody>
            {report.breakdown.map((row) => (
              <tr key={row.id} className="border-b border-border last:border-0">
                <td className="px-4 py-2">{row.label}</td>
                <td className="px-4 py-2">
                  {breakdown === "machine"
                    ? formatDurationSeconds(row.durationSeconds)
                    : formatCompactTokens(row.tokens)}
                </td>
                <td className="px-4 py-2">{formatUsdCents(row.costCents)}</td>
                <td className="px-4 py-2 text-muted-foreground">{formatShare(row.share)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

function SpendingEmptyState() {
  return (
    <div
      className="flex flex-col items-center justify-center gap-2 px-4 py-12 text-center"
      data-testid="spending-empty"
    >
      <CircleDollarSign className="size-5 text-muted-foreground" aria-hidden />
      <p className="text-[13px] text-foreground">No factory usage is recorded for this period.</p>
      <p className="max-w-sm text-[13px] text-muted-foreground">
        Change the time range or filters, or run a factory task to record spend.
      </p>
    </div>
  );
}

function breakdownLabel(breakdown: SpendingBreakdown): string {
  return SPENDING_BREAKDOWN_OPTIONS.find((option) => option.value === breakdown)?.label ?? "Workspaces";
}

function breakdownColumnLabel(breakdown: SpendingBreakdown): string {
  if (breakdown === "workspace") {
    return "Workspace";
  }
  if (breakdown === "user") {
    return "User";
  }
  if (breakdown === "model") {
    return "Model";
  }
  return "Machine type";
}
