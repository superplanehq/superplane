import { useState, type ReactNode } from "react";
import { AlertCircle, Loader2, MoreHorizontal, RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { SegmentedNav } from "@/ui/SegmentedNav";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { workOrdersPath } from "../lib/factoryPagePaths";
import {
  VELOCITY_PERIOD_OPTIONS,
  type VelocityBreakdown,
  type VelocityPeriodDays,
  type VelocityReport,
} from "../lib/factoryVelocityReport";
import { factoryCenteredSectionBodyClassName, factoryCenteredSectionHeaderClassName } from "./factoryPageLayoutStyles";
import { AUTOMATION_RUNS_BY_PERIOD } from "./velocityAutomationsMockData";
import { VelocityAutomationsTable } from "./VelocityAutomationsTable";
import { CostCard, DeliveryCard, SummaryCard, TaskTimeCard } from "./velocityCards";
import { VelocityPeopleTable } from "./VelocityPeopleTable";
import { VelocityZeroState } from "./VelocityZeroState";
import { useVelocityPageModel, type VelocityPageModel } from "./useVelocityPageModel";

const CARD_CLASSES =
  "flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-card px-6 py-16 text-center";

export function VelocityPage() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const model = useVelocityPageModel(organizationId, factoryId, factory?.onboarding);
  usePageTitle(["Velocity", factory?.name ?? "Workspace"]);

  const header = <VelocityHeader model={model} />;

  if (model.velocity.error && !model.velocity.report) {
    return renderShell(header, <VelocityErrorState onRetry={model.velocity.refetch} />);
  }

  if (model.velocity.isLoading || !model.velocity.report) {
    return renderShell(header, <VelocityLoadingState />);
  }

  if (model.velocity.isEmpty) {
    return renderShell(header, <VelocityZeroState tasksHref={workOrdersPath(organizationId, factoryKey)} />);
  }

  return renderShell(
    header,
    <VelocityReportView
      model={model}
      report={model.velocity.report}
      organizationId={organizationId}
      factoryKey={factoryKey}
    />,
  );
}

/**
 * Gray page, white cards, so the report reads as a stack of separate findings.
 * Dark mode keeps its darker page, because the factories theme paints the
 * sidebar and the cards the same color.
 */
function renderShell(header: ReactNode, body: ReactNode) {
  return (
    <div className="min-h-full bg-sidebar dark:bg-background">
      {header}
      <div className={cn(factoryCenteredSectionBodyClassName, "space-y-5 pb-10")} data-testid="factory-velocity-page">
        {body}
      </div>
    </div>
  );
}

function VelocityReportView({
  model,
  report,
  organizationId,
  factoryKey,
}: {
  model: VelocityPageModel;
  report: VelocityReport;
  organizationId: string;
  factoryKey: string;
}) {
  const [breakdown, setBreakdown] = useState<VelocityBreakdown>("origin");
  const flow = model.taskTime.flow;
  const automations = AUTOMATION_RUNS_BY_PERIOD[model.periodDays];

  return (
    <>
      <SummaryCard
        totals={report.totals}
        caption={summaryCaption(model.periodLabel, model.periodDays, Boolean(report.previous))}
        periodDays={model.periodDays}
        medianCycleHours={flow && flow.sampleSize > 0 ? flow.medianCycleHours : undefined}
        comparison={model.comparison}
      />
      <DeliveryCard
        points={report.points}
        breakdown={breakdown}
        onBreakdownChange={setBreakdown}
        intakeSeries={report.intakeSeries}
        hasOutput={report.totals.merged > 0 || report.totals.waste > 0}
      />
      {model.people.total > 0 ? (
        <VelocityPeopleTable
          people={model.people.list}
          total={model.people.total}
          periodLabel={model.periodLabel}
          emptyAuthorship={peopleEmptyAuthorship(report, model)}
          sortKey={model.people.sortKey}
          sortDirection={model.people.sortDirection}
          onSort={model.people.onSort}
          canLoadMore={model.people.canLoadMore}
          isLoadingMore={model.people.isLoadingMore}
          onLoadMore={model.people.loadMore}
        />
      ) : null}
      {automations.length > 0 ? (
        <VelocityAutomationsTable
          automations={automations}
          organizationId={organizationId}
          factoryKey={factoryKey}
          periodLabel={model.periodLabel}
        />
      ) : null}
      <div className="grid gap-5 lg:grid-cols-2">
        <TaskTimeCard flow={flow} emptyLabel={taskTimeEmptyLabel(model)} />
        <CostCard totals={report.totals} points={report.points} />
      </div>
    </>
  );
}

function summaryCaption(periodLabel: string, periodDays: VelocityPeriodDays, hasPrevious: boolean): string {
  if (hasPrevious) {
    return `${periodLabel}. Compared with the previous ${periodDays} days.`;
  }
  return `${periodLabel}. There is no earlier period to compare with yet.`;
}

/** Explains a People table that lists SuperPlane work but no manual work yet. */
function peopleEmptyAuthorship(report: VelocityReport, model: VelocityPageModel): string | undefined {
  if (report.hasPeopleCohort) return undefined;
  if (!model.integrationId) return "Connect GitHub in workspace setup to count the pull requests people created.";
  if (!model.repository) return "Select a repository in workspace setup to count the pull requests people created.";
  if (model.velocity.peopleSyncPending) {
    return "SuperPlane is collecting merges from GitHub. The Manual work column fills in after the first sync.";
  }
  return undefined;
}

function taskTimeEmptyLabel(model: VelocityPageModel): string | undefined {
  if (model.taskTime.error) return "We could not load task time.";
  if (model.taskTime.isLoading) return "Loading task time…";
  return undefined;
}

function VelocityHeader({ model }: { model: VelocityPageModel }) {
  return (
    <>
      <VelocityHeaderBar model={model} />
      {model.sync.isSyncing ? (
        <div className={cn(factoryCenteredSectionBodyClassName, "pt-1 pb-4")}>
          <VelocitySyncProgress />
        </div>
      ) : null}
    </>
  );
}

function VelocityHeaderBar({ model }: { model: VelocityPageModel }) {
  return (
    <WorkspacePageHeader
      className={factoryCenteredSectionHeaderClassName}
      title="Velocity"
      subtitle={
        model.repository
          ? `What ${model.repository} ships, how long the work takes, and what it costs.`
          : "What this workspace ships, how long the work takes, and what it costs."
      }
      actions={
        <div className="flex items-center gap-2">
          <SegmentedNav
            ariaLabel="Velocity period in days"
            size="xs"
            value={String(model.periodDays)}
            onValueChange={(value) => {
              const next = Number(value);
              if (next === 14 || next === 30) model.setPeriodDays(next as VelocityPeriodDays);
            }}
            options={VELOCITY_PERIOD_OPTIONS}
          />
          <VelocityOverflowMenu model={model} />
        </div>
      }
    />
  );
}

/**
 * Shows that a sync is in flight.
 *
 * The sync reads sixty days of history in a background worker that reports no
 * progress of its own, so the bar is indeterminate: it says work is happening
 * without claiming to know how much is left.
 */
function VelocitySyncProgress() {
  return (
    <div
      className="h-1 w-full overflow-hidden rounded-full bg-primary/15"
      role="progressbar"
      aria-label="Reading merges from GitHub"
      data-testid="velocity-sync-progress"
    >
      <div className="sp-indeterminate-bar h-full w-1/4 rounded-full bg-primary" />
    </div>
  );
}

/**
 * Merge counts come from a background sync rather than from GitHub at request
 * time, so a merge made moments ago is not on the page yet. This asks for a
 * fresh read instead of leaving the user to wait for the next scheduled sync.
 */
function VelocityOverflowMenu({ model }: { model: VelocityPageModel }) {
  if (model.sync.isUnavailable) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="text-muted-foreground hover:bg-accent hover:text-foreground"
          aria-label="Velocity menu"
          data-testid="velocity-overflow-menu"
        >
          <MoreHorizontal className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={6} className="min-w-[12rem] rounded-xl border-border p-1 shadow-lg">
        <DropdownMenuItem
          className="cursor-pointer rounded-md px-2 py-1.5 text-[13px] [&_svg]:size-3.5"
          onClick={model.sync.start}
          disabled={model.sync.isSyncing}
          title={refreshDataTitle(model.velocity.syncedAt)}
          data-testid="velocity-refresh-data"
        >
          <RefreshCw className={cn(model.sync.isSyncing && "animate-spin")} aria-hidden />
          Refresh data
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function refreshDataTitle(syncedAt?: Date): string {
  const action = "Read the merges of the last 60 days from GitHub.";
  if (!syncedAt) return action;
  return `${action} Last synced ${syncedAt.toLocaleString()}.`;
}

function VelocityLoadingState() {
  return (
    <div className={CARD_CLASSES} data-testid="velocity-loading-state">
      <Loader2 className="size-5 animate-spin text-muted-foreground" aria-hidden />
      <p className="text-[13px] text-muted-foreground">Loading velocity…</p>
    </div>
  );
}

function VelocityErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className={CARD_CLASSES} data-testid="velocity-error-state">
      <AlertCircle className="size-5 text-destructive" aria-hidden />
      <div className="space-y-1">
        <p className="text-[13px] font-medium text-foreground">We could not load velocity.</p>
        <p className="text-[12px] text-muted-foreground">Check your network and try again.</p>
      </div>
      <Button type="button" variant="outline" size="sm" onClick={onRetry} data-testid="velocity-error-retry">
        Retry
      </Button>
    </div>
  );
}
