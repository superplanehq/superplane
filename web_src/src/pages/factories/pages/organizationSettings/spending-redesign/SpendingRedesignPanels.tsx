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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Separator } from "@/ui/separator";

import { pickVelocityAxisTicks } from "../../../lib/velocityAxisTicks";
import { formatUsdCents } from "../../../lib/workOrderUsage";
import { factoryCardClassName } from "../../factoryPageLayoutStyles";
import type { SpendingCreditSnapshot } from "./spendingRedesignMocks";
import {
  EMPTY_SPENDING_FILTERS,
  formatFilterTriggerLabel,
  formatShare,
  hasActiveSpendingFilters,
  MACHINE_BREAKDOWN_OPTIONS,
  MODEL_BREAKDOWN_OPTIONS,
  type SpendingBreakdown,
  type SpendingCatalogItem,
  type SpendingCatalogs,
  type SpendingFilters,
  type SpendingReport,
  type SpendingUsageKind,
} from "./spendingRedesignLib";

const SERIES_COLORS = ["#6366f1", "#10b981", "#f59e0b", "#3b82f6", "#8b5cf6", "#94a3b8"];
const ALL_FILTER_VALUE = "all";

export function SpendingUsageSection({
  kind,
  catalogs,
  filters,
  breakdown,
  report,
  onChange,
  onBreakdownChange,
}: {
  kind: SpendingUsageKind;
  catalogs: SpendingCatalogs;
  filters: SpendingFilters;
  breakdown: SpendingBreakdown;
  report: SpendingReport;
  onChange: (filters: SpendingFilters) => void;
  onBreakdownChange: (value: SpendingBreakdown) => void;
}) {
  const copy = usageCopy(kind);
  const prefix = copy.testIdPrefix;
  const empty = report.totals.costCents === 0;

  return (
    <section className="flex flex-col gap-5" data-testid={`${prefix}-usage`}>
      <div>
        <h2 className="workspace-section-title">{copy.title}</h2>
        <p className="mt-0.5 text-[12px] text-muted-foreground">{copy.description}</p>
        <div className="mt-3">
          <SpendingFilterBar
            breakdown={breakdown}
            breakdownOptions={copy.breakdownOptions}
            catalogs={catalogs}
            filters={filters}
            kind={kind}
            testIdPrefix={prefix}
            onBreakdownChange={onBreakdownChange}
            onChange={onChange}
          />
        </div>
      </div>
      <SpendingChartCard
        breakdown={breakdown}
        breakdownOptions={copy.breakdownOptions}
        empty={empty}
        emptyMessage={copy.emptyMessage}
        report={report}
        testId={`${prefix}-chart`}
      />
      <SpendingBreakdownCard
        breakdown={breakdown}
        empty={empty}
        emptyMessage={copy.emptyMessage}
        report={report}
        testId={`${prefix}-breakdown`}
      />
    </section>
  );
}

function SpendingFilterBar({
  catalogs,
  filters,
  kind,
  breakdown,
  breakdownOptions,
  testIdPrefix,
  onChange,
  onBreakdownChange,
}: {
  catalogs: SpendingCatalogs;
  filters: SpendingFilters;
  kind: SpendingUsageKind;
  breakdown: SpendingBreakdown;
  breakdownOptions: Array<{ value: SpendingBreakdown; label: string }>;
  testIdPrefix: string;
  onChange: (filters: SpendingFilters) => void;
  onBreakdownChange: (value: SpendingBreakdown) => void;
}) {
  const filtersActive = hasActiveSpendingFilters(filters);

  return (
    <div className="flex flex-wrap items-center gap-2" data-testid={`${testIdPrefix}-filter-bar`}>
      <SpendingFilterMenu
        allLabel="All users"
        items={catalogs.users}
        selected={filters.userId}
        testId={`${testIdPrefix}-filter-users`}
        onSelect={(userId) => onChange({ ...filters, userId })}
      />
      <SpendingFilterMenu
        allLabel="All workspaces"
        items={catalogs.workspaces}
        selected={filters.workspaceId}
        testId={`${testIdPrefix}-filter-workspaces`}
        onSelect={(workspaceId) => onChange({ ...filters, workspaceId })}
      />
      {kind === "model" ? (
        <SpendingFilterMenu
          allLabel="All models"
          items={catalogs.models}
          selected={filters.model}
          testId={`${testIdPrefix}-filter-models`}
          onSelect={(model) => onChange({ ...filters, model })}
        />
      ) : (
        <SpendingFilterMenu
          allLabel="All machine types"
          items={catalogs.machines}
          selected={filters.machineType}
          testId={`${testIdPrefix}-filter-machines`}
          onSelect={(machineType) => onChange({ ...filters, machineType })}
        />
      )}
      {filtersActive ? (
        <Button
          variant="ghost"
          size="sm"
          data-testid={`${testIdPrefix}-clear-filters`}
          onClick={() => onChange(EMPTY_SPENDING_FILTERS)}
        >
          Clear filters
        </Button>
      ) : null}
      <Separator
        orientation="vertical"
        className="mx-2 h-7 w-px self-center bg-slate-950/20 dark:bg-gray-600/70"
        data-testid={`${testIdPrefix}-group-by-separator`}
      />
      <SpendingGroupBy
        breakdown={breakdown}
        breakdownOptions={breakdownOptions}
        testId={`${testIdPrefix}-group-by`}
        onBreakdownChange={onBreakdownChange}
      />
    </div>
  );
}

function SpendingGroupBy({
  breakdown,
  breakdownOptions,
  testId,
  onBreakdownChange,
}: {
  breakdown: SpendingBreakdown;
  breakdownOptions: Array<{ value: SpendingBreakdown; label: string }>;
  testId: string;
  onBreakdownChange: (value: SpendingBreakdown) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" aria-label="Group by" data-testid={testId}>
          Group by {breakdownLabel(breakdown, breakdownOptions)}
          <ChevronDown className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-48">
        <DropdownMenuRadioGroup
          value={breakdown}
          onValueChange={(value) => {
            const next = breakdownOptions.find((option) => option.value === value);
            if (next) {
              onBreakdownChange(next.value);
            }
          }}
        >
          {breakdownOptions.map((option) => (
            <DropdownMenuRadioItem key={option.value} value={option.value}>
              {option.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function SpendingFilterMenu({
  allLabel,
  items,
  selected,
  testId,
  onSelect,
}: {
  allLabel: string;
  items: SpendingCatalogItem[];
  selected: string;
  testId: string;
  onSelect: (id: string) => void;
}) {
  const selectedLabel = items.find((item) => item.id === selected)?.label;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" data-testid={testId}>
          {formatFilterTriggerLabel(allLabel, selectedLabel)}
          <ChevronDown className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-48">
        <DropdownMenuRadioGroup
          value={selected || ALL_FILTER_VALUE}
          onValueChange={(value) => onSelect(value === ALL_FILTER_VALUE ? "" : value)}
        >
          <DropdownMenuRadioItem value={ALL_FILTER_VALUE}>{allLabel}</DropdownMenuRadioItem>
          {items.map((item) => (
            <DropdownMenuRadioItem key={item.id} value={item.id}>
              {item.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
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

function SpendingChartCard({
  breakdown,
  breakdownOptions,
  empty,
  emptyMessage,
  report,
  testId,
}: {
  breakdown: SpendingBreakdown;
  breakdownOptions: Array<{ value: SpendingBreakdown; label: string }>;
  empty: boolean;
  emptyMessage: string;
  report: SpendingReport;
  testId: string;
}) {
  return (
    <section className={factoryCardClassName} data-testid={testId}>
      <div className="px-4 pt-4">
        <h3 className="workspace-section-title">Spend over time</h3>
        <p className="mt-0.5 text-[12px] text-muted-foreground">
          Stacked by {breakdownLabel(breakdown, breakdownOptions).toLowerCase()}.
        </p>
      </div>
      {empty ? (
        <SpendingEmptyState message={emptyMessage} />
      ) : (
        <div className="px-2 pb-4 pt-2">
          <SpendingBarChart report={report} />
        </div>
      )}
    </section>
  );
}

function SpendingBreakdownCard({
  breakdown,
  empty,
  emptyMessage,
  report,
  testId,
}: {
  breakdown: SpendingBreakdown;
  empty: boolean;
  emptyMessage: string;
  report: SpendingReport;
  testId: string;
}) {
  return (
    <section className={factoryCardClassName} data-testid={testId}>
      <div className="px-4 pt-4">
        <h3 className="workspace-section-title">Breakdown</h3>
        <p className="mt-0.5 text-[12px] text-muted-foreground">
          {breakdown === "user"
            ? "Spend is grouped by the task owner."
            : `One row per ${breakdownColumnLabel(breakdown).toLowerCase()} in this range.`}
        </p>
      </div>
      {empty ? (
        <SpendingEmptyState message={emptyMessage} />
      ) : (
        <table className="mt-2 w-full text-left text-[13px]">
          <thead>
            <tr className="border-b border-border text-muted-foreground">
              <th className="px-4 py-2 font-medium">{breakdownColumnLabel(breakdown)}</th>
              <th className="px-4 py-2 font-medium">Spend</th>
              <th className="px-4 py-2 font-medium">Share</th>
            </tr>
          </thead>
          <tbody>
            {report.breakdown.map((row) => (
              <tr key={row.id} className="border-b border-border last:border-0">
                <td className="px-4 py-2">{row.label}</td>
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

function SpendingEmptyState({ message }: { message: string }) {
  return (
    <div
      className="flex flex-col items-center justify-center gap-2 px-4 py-12 text-center"
      data-testid="spending-empty"
    >
      <CircleDollarSign className="size-5 text-muted-foreground" aria-hidden />
      <p className="text-[13px] text-foreground">{message}</p>
      <p className="max-w-sm text-[13px] text-muted-foreground">
        Change the time range or filters, or run a factory task to record spend.
      </p>
    </div>
  );
}

function usageCopy(kind: SpendingUsageKind): {
  title: string;
  description: string;
  emptyMessage: string;
  testIdPrefix: string;
  breakdownOptions: Array<{ value: SpendingBreakdown; label: string }>;
} {
  if (kind === "model") {
    return {
      title: "Model usage",
      description: "Estimated spend in dollars for SuperPlane-hosted models and your keys.",
      emptyMessage: "No model usage is recorded for this period.",
      testIdPrefix: "spending-model",
      breakdownOptions: MODEL_BREAKDOWN_OPTIONS,
    };
  }
  return {
    title: "VM usage",
    description: "Estimated spend in dollars for SuperPlane runner machines.",
    emptyMessage: "No VM usage is recorded for this period.",
    testIdPrefix: "spending-vm",
    breakdownOptions: MACHINE_BREAKDOWN_OPTIONS,
  };
}

function breakdownLabel(
  breakdown: SpendingBreakdown,
  options: Array<{ value: SpendingBreakdown; label: string }>,
): string {
  return options.find((option) => option.value === breakdown)?.label ?? "Workspaces";
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
