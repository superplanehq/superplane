import { CalendarDays } from "lucide-react";
import type { DateRange } from "react-day-picker";

import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { Calendar } from "@/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { SegmentedNav } from "@/ui/SegmentedNav";

import { WorkspacePageHeader } from "../../../layout/WorkspacePageHeader";
import {
  factoryCenteredSectionBodyClassName,
  factoryCenteredSectionHeaderClassName,
} from "../../factoryPageLayoutStyles";
import type { SpendingCreditSnapshot } from "./spendingRedesignMocks";
import {
  formatSpendingRangeCaption,
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
import { SpendingKpiRow, SpendingUsageSection } from "./SpendingRedesignPanels";
import { useSpendingRedesignPageModel, type SpendingRedesignControlledState } from "./useSpendingRedesignPageModel";

export interface SpendingRedesignPageProps extends SpendingRedesignControlledState {
  events?: SpendingUsageEvent[];
  catalogs: SpendingCatalogs;
  credit: SpendingCreditSnapshot;
  now?: Date;
  initialPeriod?: SpendingPeriodPreset;
  initialModelFilters?: SpendingFilters;
  initialMachineFilters?: SpendingFilters;
  initialModelBreakdown?: SpendingBreakdown;
  initialMachineBreakdown?: SpendingBreakdown;
  initialCustomRange?: SpendingDateRange;
  isLoading?: boolean;
  errorMessage?: string;
}

/**
 * Organization Spending explorer.
 *
 * Storybook passes ledger events. Production passes server-built reports from
 * the spending-report API.
 */
export function SpendingRedesignPage(props: SpendingRedesignPageProps) {
  const { credit, catalogs, isLoading = false, errorMessage, ...modelArgs } = props;
  usePageTitle(["Spending"]);
  const view = useSpendingRedesignPageModel({ ...modelArgs, catalogs });
  const metrics = spendingMetricCopy(view.rangeTotals);

  if (errorMessage) {
    return (
      <div className="min-h-full bg-sidebar p-6 dark:bg-background" data-testid="spending-redesign-page">
        <p className="text-[13px] text-destructive">{errorMessage}</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="min-h-full bg-sidebar p-6 dark:bg-background" data-testid="spending-redesign-page">
        <p className="text-[13px] text-muted-foreground">Loading spending...</p>
      </div>
    );
  }

  return (
    <div className="min-h-full bg-sidebar dark:bg-background" data-testid="spending-redesign-page">
      <WorkspacePageHeader
        className={factoryCenteredSectionHeaderClassName}
        title="Spending"
        subtitle="Review factory token usage, VM time, and estimated spend for this organization."
        actions={
          <SpendingPeriodControls
            customOpen={view.customOpen}
            customRange={view.range}
            period={view.period}
            onCustomOpenChange={view.setCustomOpen}
            onCustomRangeChange={view.setCustomRange}
            onPeriodChange={view.handlePeriodChange}
          />
        }
      />
      <div className={cn(factoryCenteredSectionBodyClassName, "flex flex-col gap-5 pb-10")}>
        <SpendingKpiRow credit={credit} metrics={metrics} rangeCaption={formatSpendingRangeCaption(view.range)} />
        <SpendingUsageSection
          breakdown={view.modelBreakdown}
          catalogs={catalogs}
          filters={view.modelFilters}
          kind="model"
          report={view.modelReport}
          onBreakdownChange={view.setModelBreakdown}
          onChange={view.setModelFilters}
        />
        <SpendingUsageSection
          breakdown={view.machineBreakdown}
          catalogs={catalogs}
          filters={view.machineFilters}
          kind="compute"
          report={view.machineReport}
          onBreakdownChange={view.setMachineBreakdown}
          onChange={view.setMachineFilters}
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
