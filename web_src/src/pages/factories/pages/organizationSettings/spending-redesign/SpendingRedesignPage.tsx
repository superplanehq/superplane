import { Loader2 } from "lucide-react";

import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";

import { WorkspacePageHeader } from "../../../layout/WorkspacePageHeader";
import {
  factoryCenteredSectionBodyClassName,
  factoryCenteredSectionHeaderClassName,
} from "../../factoryPageLayoutStyles";
import type { SpendingCreditSnapshot } from "./spendingRedesignMocks";
import {
  formatSpendingRangeCaption,
  spendingMetricCopy,
  type SpendingBreakdown,
  type SpendingCatalogs,
  type SpendingDateRange,
  type SpendingFilters,
  type SpendingPeriodPreset,
  type SpendingUsageEvent,
} from "./spendingRedesignLib";
import { SpendingPeriodControls } from "./SpendingPeriodControls";
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
  // resolves. Once we have them, keep rendering them (and swap in the small
  // top-right indicator below) instead of dropping back to the full-page
  // loading state on every refetch, including on remounts that reuse a
  // cached report.
  //
  // Require *both* reports so that a partially resolved first load (the model
  // query settling before the compute query, or vice versa) does not suppress
  // the full-page loading state and render the still-loading section as empty
  // data behind a background-refresh indicator.
  const hasReport = Boolean(props.modelReport && props.machineReport);
  const showFullPageLoading = isLoading && !hasReport;

  if (errorMessage) {
    return (
      <div className="min-h-full bg-sidebar p-6 dark:bg-background" data-testid="spending-redesign-page">
        <p className="text-[13px] text-destructive">{errorMessage}</p>
      </div>
    );
  }

  if (showFullPageLoading) {
    return <SpendingPageLoading />;
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
 * First-load placeholder for the spending report. Centered in the settings
 * pane so it does not read as leftover copy in the top-left corner.
 */
function SpendingPageLoading() {
  return (
    <div
      className="flex h-full min-h-0 flex-1 items-center justify-center bg-sidebar dark:bg-background"
      data-testid="spending-redesign-page"
    >
      <div data-testid="spending-page-loading" role="status">
        <Loader2 className="size-8 animate-spin text-muted-foreground" aria-hidden />
        <span className="sr-only">Loading spending</span>
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
