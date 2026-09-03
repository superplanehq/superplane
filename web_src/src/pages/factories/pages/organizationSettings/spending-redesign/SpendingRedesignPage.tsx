import { useMemo, useState } from "react";
import { CalendarDays } from "lucide-react";
import type { DateRange } from "react-day-picker";

import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { Calendar } from "@/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { SegmentedNav } from "@/ui/SegmentedNav";
import { HostedCreditSummary } from "@/pages/organization/settings/HostedCreditSummary";

import { WorkspacePageHeader } from "../../../layout/WorkspacePageHeader";
import {
  factoryCardClassName,
  factoryCenteredSectionBodyClassName,
  factoryCenteredSectionHeaderClassName,
} from "../../factoryPageLayoutStyles";
import type { SpendingCreditSnapshot } from "./spendingRedesignMocks";
import {
  buildSpendingReport,
  EMPTY_SPENDING_FILTERS,
  formatSpendingRangeCaption,
  hasActiveSpendingFilters,
  rangeForPreset,
  rangeFromCustomDays,
  spendingMetricCopy,
  SPENDING_PERIOD_OPTIONS,
  type SpendingBreakdown,
  type SpendingCatalogs,
  type SpendingDateRange,
  type SpendingFilters,
  type SpendingPeriodPreset,
  type SpendingUsageEvent,
} from "./spendingRedesignLib";
import { SpendingBreakdownCard, SpendingChartCard, SpendingFilterBar, SpendingKpiRow } from "./SpendingRedesignPanels";

export interface SpendingRedesignPageProps {
  events: SpendingUsageEvent[];
  catalogs: SpendingCatalogs;
  credit: SpendingCreditSnapshot;
  now: Date;
  initialPeriod?: SpendingPeriodPreset;
  initialFilters?: SpendingFilters;
  initialBreakdown?: SpendingBreakdown;
  initialCustomRange?: SpendingDateRange;
}

/**
 * Organization Spending explorer (Storybook-only).
 *
 * Time range, user, workspace, model, and machine filters slice the same
 * ledger the live Spending page reads: model tokens and runner VM time, with
 * estimated dollars. Task-owner is the user dimension because usage rows do
 * not store a user id.
 */
export function SpendingRedesignPage({
  events,
  catalogs,
  credit,
  now,
  initialPeriod = "month",
  initialFilters = EMPTY_SPENDING_FILTERS,
  initialBreakdown = "workspace",
  initialCustomRange,
}: SpendingRedesignPageProps) {
  usePageTitle(["Spending"]);
  const [period, setPeriod] = useState<SpendingPeriodPreset>(initialPeriod);
  const [customRange, setCustomRange] = useState<SpendingDateRange | undefined>(initialCustomRange);
  const [customOpen, setCustomOpen] = useState(initialPeriod === "custom");
  const [filters, setFilters] = useState<SpendingFilters>(initialFilters);
  const [breakdown, setBreakdown] = useState<SpendingBreakdown>(initialBreakdown);

  const range = useMemo(() => {
    if (period === "custom") {
      return customRange ?? rangeForPreset("week", now);
    }
    return rangeForPreset(period, now);
  }, [customRange, now, period]);

  const report = useMemo(
    () => buildSpendingReport(events, range, filters, breakdown, catalogs),
    [breakdown, catalogs, events, filters, range],
  );
  const metrics = spendingMetricCopy(report.totals);
  const filtersActive = hasActiveSpendingFilters(filters);

  const handlePeriodChange = (value: string) => {
    const next = value as SpendingPeriodPreset;
    if (next === "custom") {
      setPeriod("custom");
      setCustomOpen(true);
      if (!customRange) {
        setCustomRange(rangeForPreset("week", now));
      }
      return;
    }
    setPeriod(next);
    setCustomOpen(false);
  };

  return (
    <div className="min-h-full bg-sidebar dark:bg-background" data-testid="spending-redesign-page">
      <WorkspacePageHeader
        className={factoryCenteredSectionHeaderClassName}
        title="Spending"
        subtitle="Review factory token usage, VM time, and estimated spend for this organization."
        actions={
          <SpendingPeriodControls
            customOpen={customOpen}
            customRange={range}
            period={period}
            onCustomOpenChange={setCustomOpen}
            onCustomRangeChange={(next) => {
              setCustomRange(next);
              setPeriod("custom");
            }}
            onPeriodChange={handlePeriodChange}
          />
        }
        belowRow={
          <SpendingFilterBar
            catalogs={catalogs}
            filters={filters}
            filtersActive={filtersActive}
            onChange={setFilters}
          />
        }
      />
      <div className={cn(factoryCenteredSectionBodyClassName, "flex flex-col gap-5 pb-10")}>
        <SpendingKpiRow credit={credit} metrics={metrics} rangeCaption={formatSpendingRangeCaption(range)} />
        <SpendingChartCard breakdown={breakdown} report={report} />
        <SpendingBreakdownCard breakdown={breakdown} onBreakdownChange={setBreakdown} report={report} />
        <HostedCreditSummary
          remainingCreditCents={credit.remainingCreditCents}
          grantTotalCents={credit.grantTotalCents}
          superplaneGrantCents={credit.superplaneGrantCents}
          purchasedCreditCents={credit.purchasedCreditCents}
          hostedBilledCents={credit.hostedBilledCents}
          remainingCreditWarning={credit.remainingCreditWarning}
          billingEnabled={credit.billingEnabled}
          hasBillingCustomer={credit.hasBillingCustomer}
          canManageBilling
          cardClassName={`${factoryCardClassName} p-4`}
          labelClassName="workspace-section-label"
          valueClassName="workspace-page-title mt-1"
        />
      </div>
    </div>
  );
}

function SpendingPeriodControls({
  period,
  customRange,
  customOpen,
  onPeriodChange,
  onCustomOpenChange,
  onCustomRangeChange,
}: {
  period: SpendingPeriodPreset;
  customRange: SpendingDateRange;
  customOpen: boolean;
  onPeriodChange: (value: string) => void;
  onCustomOpenChange: (open: boolean) => void;
  onCustomRangeChange: (range: SpendingDateRange) => void;
}) {
  const selected: DateRange = {
    from: customRange.start,
    to: new Date(customRange.end.getTime() - 1),
  };

  return (
    <div className="flex flex-wrap items-center justify-end gap-2">
      <SegmentedNav
        ariaLabel="Spending time range"
        options={SPENDING_PERIOD_OPTIONS}
        size="xs"
        value={period}
        onValueChange={onPeriodChange}
      />
      <Popover open={customOpen} onOpenChange={onCustomOpenChange}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            aria-label="Select a custom date range"
            data-testid="spending-custom-range"
          >
            <CalendarDays className="size-3.5" aria-hidden />
            {period === "custom" ? formatSpendingRangeCaption(customRange) : "Custom range"}
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" className="w-auto p-3">
          <Calendar
            mode="range"
            numberOfMonths={2}
            selected={selected}
            onSelect={(next) => {
              if (!next?.from) {
                return;
              }
              onCustomRangeChange(rangeFromCustomDays(next.from, next.to ?? next.from));
            }}
          />
        </PopoverContent>
      </Popover>
    </div>
  );
}
