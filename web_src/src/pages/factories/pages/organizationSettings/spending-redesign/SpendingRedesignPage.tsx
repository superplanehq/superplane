import { useState } from "react";
import { CalendarDays, ChevronDown, Loader2 } from "lucide-react";
import type { DateRange } from "react-day-picker";

import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { Calendar } from "@/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";

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
  spendingPeriodTriggerLabel,
  SPENDING_PERIOD_PRESETS,
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
  /**
   * A background refetch is in flight (for example, the report is
   * revalidating after a return visit to this tab). Unlike `isLoading`,
   * this can be true while data from a previous load is already on screen.
   */
  isFetching?: boolean;
  errorMessage?: string;
}

/**
 * Organization Spending explorer.
 *
 * Storybook passes ledger events. Production passes server-built reports from
 * the spending-report API.
 */
export function SpendingRedesignPage(props: SpendingRedesignPageProps) {
  const { credit, catalogs, isLoading = false, isFetching = false, errorMessage, ...modelArgs } = props;
  usePageTitle(["Spending"]);
  const view = useSpendingRedesignPageModel({ ...modelArgs, catalogs });
  const metrics = spendingMetricCopy(view.rangeTotals);
  // Reports from a real query are only present once the first fetch
  // resolves. Once we have one, keep rendering it (and swap in the small
  // top-right indicator below) instead of dropping back to the full-page
  // loading state on every refetch, including on remounts that reuse a
  // cached report.
  const hasReport = Boolean(props.modelReport || props.machineReport);
  const showFullPageLoading = isLoading && !hasReport;

  if (errorMessage) {
    return (
      <div className="min-h-full bg-sidebar p-6 dark:bg-background" data-testid="spending-redesign-page">
        <p className="text-[13px] text-destructive">{errorMessage}</p>
      </div>
    );
  }

  if (showFullPageLoading) {
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
          <>
            {isFetching && hasReport ? <SpendingRefetchIndicator /> : null}
            <SpendingPeriodControls
              customOpen={view.customOpen}
              customRange={view.range}
              period={view.period}
              onCustomOpenChange={view.setCustomOpen}
              onCustomRangeChange={view.setCustomRange}
              onPeriodChange={view.handlePeriodChange}
            />
          </>
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

/**
 * Quiet indicator shown in the top-right of the header while the spending
 * report revalidates over data that is already on screen. Deliberately
 * small and unobtrusive: it must not compete with the loading state used
 * for the true first load.
 */
function SpendingRefetchIndicator() {
  return (
    <span
      className="flex items-center gap-1.5 text-[12px] text-muted-foreground"
      data-testid="spending-refetch-indicator"
      role="status"
      aria-live="polite"
    >
      <Loader2 className="size-3.5 animate-spin" aria-hidden />
      Refreshing spending...
    </span>
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
  const committed: DateRange = {
    from: customRange.start,
    to: new Date(customRange.end.getTime() - 1),
  };
  const [draftRange, setDraftRange] = useState<DateRange | undefined>();
  const selected = draftRange ?? committed;
  const triggerLabel = spendingPeriodTriggerLabel(period, customRange);

  return (
    <Popover
      open={customOpen}
      onOpenChange={(open) => {
        if (!open) {
          setDraftRange(undefined);
        }
        onCustomOpenChange(open);
      }}
    >
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          aria-expanded={customOpen}
          aria-haspopup="dialog"
          aria-label={`Spending period, ${triggerLabel}`}
          data-testid="spending-period"
        >
          <CalendarDays className="size-3.5" aria-hidden />
          {triggerLabel}
          <ChevronDown className="size-3.5" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-auto overflow-visible p-0" data-testid="spending-period-picker">
        <div className="flex flex-col sm:flex-row">
          <div
            role="radiogroup"
            aria-label="Spending period"
            className="flex flex-col gap-0.5 border-b p-2 sm:w-44 sm:border-r sm:border-b-0"
          >
            {SPENDING_PERIOD_PRESETS.map((option) => {
              const isActive = period === option.value;
              return (
                <button
                  key={option.value}
                  type="button"
                  role="radio"
                  aria-checked={isActive}
                  className={cn(
                    "rounded-md px-2.5 py-1.5 text-left text-[13px]",
                    isActive
                      ? "bg-accent font-medium text-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-foreground",
                  )}
                  onClick={() => onPeriodChange(option.value)}
                >
                  {option.label}
                </button>
              );
            })}
          </div>
          <Calendar
            className="p-2 [--cell-size:2rem]"
            classNames={{ root: "rdp-root w-[16.5rem]" }}
            defaultMonth={selected.from}
            mode="range"
            selected={selected}
            onSelect={(next) => {
              if (!next?.from) {
                return;
              }
              setDraftRange(next);
              if (!next.to) {
                return;
              }
              onCustomRangeChange(rangeFromCustomDays(next.from, next.to));
            }}
          />
        </div>
      </PopoverContent>
    </Popover>
  );
}
