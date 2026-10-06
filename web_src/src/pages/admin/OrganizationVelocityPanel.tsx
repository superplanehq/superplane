import { useState, type ReactNode } from "react";

import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { TooltipProvider } from "@/ui/tooltip";
import { SegmentedNav } from "@/ui/SegmentedNav";

import { factoryVelocityPeriodLabel } from "@/pages/factories/lib/factoryVelocityFlow";
import {
  VELOCITY_PERIOD_OPTIONS,
  hasVelocityOutput,
  isVelocityPeriodDays,
  toVelocityReport,
  type VelocityBreakdown,
  type VelocityPeriodDays,
  type VelocityReport,
} from "@/pages/factories/lib/factoryVelocityReport";
import { VelocityAutomationsTable } from "@/pages/factories/pages/VelocityAutomationsTable";
import {
  CostCard,
  DeliveryCard,
  SummaryCard,
  TaskCostCard,
  type VelocityComparison,
} from "@/pages/factories/pages/velocityCards";

import { formatDateTime } from "./formatDate";
import { useAdminOrganizationVelocity, type AdminVelocityFactory } from "./useAdminOrganizationVelocity";

const NO_WORKSPACES = "This organization has no workspaces.";
const NO_VELOCITY = "There is no velocity in this period.";
const LOAD_ERROR = "Could not load velocity.";
const REFRESH_ERROR = "Could not refresh velocity.";

export function OrganizationVelocityPanel({ orgId }: { orgId: string }) {
  const [periodDays, setPeriodDays] = useState<VelocityPeriodDays>(30);
  const [requestedFactoryId, setRequestedFactoryId] = useState<string | undefined>();
  const velocity = useAdminOrganizationVelocity(orgId, periodDays, requestedFactoryId);

  if (velocity.isLoading && !velocity.data) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading velocity.</Text>;
  }

  if (velocity.error && !velocity.data) {
    return <VelocityLoadError onRetry={() => void velocity.refetch()} />;
  }

  return (
    <LoadedVelocity
      factories={velocity.data?.factories ?? []}
      factoryId={requestedFactoryId ?? velocity.data?.factoryId}
      report={velocity.data ? toVelocityReport(velocity.data) : undefined}
      periodDays={periodDays}
      onPeriodDays={setPeriodDays}
      onFactory={setRequestedFactoryId}
      refreshError={refreshErrorFor(velocity, () => void velocity.refetch())}
    />
  );
}

function refreshErrorFor(velocity: { isError: boolean; data?: unknown; dataUpdatedAt: number }, onRetry: () => void) {
  if (!velocity.isError || !velocity.data) {
    return null;
  }
  return <VelocityRefreshError loadedAt={velocity.dataUpdatedAt} onRetry={onRetry} />;
}

function VelocityLoadError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex flex-col items-start gap-3">
      <Text className="text-sm text-gray-500 dark:text-gray-400">{LOAD_ERROR}</Text>
      <Button type="button" variant="outline" size="sm" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

function LoadedVelocity({
  factories,
  factoryId,
  report,
  periodDays,
  onPeriodDays,
  onFactory,
  refreshError,
}: {
  factories: AdminVelocityFactory[];
  factoryId?: string;
  report?: VelocityReport;
  periodDays: VelocityPeriodDays;
  onPeriodDays: (days: VelocityPeriodDays) => void;
  onFactory: (factoryId: string) => void;
  refreshError: ReactNode;
}) {
  if (factories.length === 0) {
    return (
      <div className="flex flex-col items-start gap-3">
        {refreshError}
        <Text className="text-sm text-gray-500 dark:text-gray-400">{NO_WORKSPACES}</Text>
      </div>
    );
  }

  const selected = selectedFactory(factories, factoryId);

  return (
    <TooltipProvider delayDuration={150}>
      <div className="space-y-5" data-testid="admin-organization-velocity">
        {refreshError}
        <div className="flex flex-wrap items-center justify-between gap-3">
          <WorkspaceControl factories={factories} selectedId={selected?.id} onSelect={onFactory} />
          <PeriodControl periodDays={periodDays} onPeriodDays={onPeriodDays} />
        </div>
        {report && hasVelocityOutput(report) ? (
          <VelocityCards report={report} periodDays={periodDays} />
        ) : (
          <Text className="text-sm text-gray-500 dark:text-gray-400">{NO_VELOCITY}</Text>
        )}
      </div>
    </TooltipProvider>
  );
}

function VelocityRefreshError({ loadedAt, onRetry }: { loadedAt: number; onRetry: () => void }) {
  const loadedLabel = loadedAt > 0 ? formatDateTime(new Date(loadedAt).toISOString()) : "";

  return (
    <div className="flex flex-col items-start gap-2" role="alert" data-testid="admin-velocity-refresh-error">
      <Text className="text-sm text-red-600 dark:text-red-400">{REFRESH_ERROR}</Text>
      {loadedLabel ? (
        <Text className="text-sm text-gray-500 dark:text-gray-400">Last loaded {loadedLabel}.</Text>
      ) : null}
      <Button type="button" variant="outline" size="sm" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

function WorkspaceControl({
  factories,
  selectedId,
  onSelect,
}: {
  factories: AdminVelocityFactory[];
  selectedId?: string;
  onSelect: (factoryId: string) => void;
}) {
  const selected = factories.find((factory) => factory.id === selectedId) ?? factories[0];
  if (factories.length === 1) {
    return <p className="text-sm font-medium text-gray-800 dark:text-gray-100">{selected?.name}</p>;
  }

  return (
    <Select value={selected?.id} onValueChange={onSelect}>
      <SelectTrigger aria-label="Workspace" className="h-9 w-56" data-testid="admin-velocity-workspace">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {factories.map((factory) => (
          <SelectItem key={factory.id} value={factory.id}>
            {factory.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function PeriodControl({
  periodDays,
  onPeriodDays,
}: {
  periodDays: VelocityPeriodDays;
  onPeriodDays: (days: VelocityPeriodDays) => void;
}) {
  return (
    <SegmentedNav
      ariaLabel="Velocity period in days"
      size="xs"
      value={String(periodDays)}
      onValueChange={(value) => {
        const next = Number(value);
        if (isVelocityPeriodDays(next)) {
          onPeriodDays(next);
        }
      }}
      options={VELOCITY_PERIOD_OPTIONS}
    />
  );
}

function VelocityCards({ report, periodDays }: { report: VelocityReport; periodDays: VelocityPeriodDays }) {
  const [breakdown, setBreakdown] = useState<VelocityBreakdown>("origin");
  const periodLabel = factoryVelocityPeriodLabel(periodDays);

  return (
    <>
      <SummaryCard
        totals={report.totals}
        caption={summaryCaption(periodLabel, periodDays, Boolean(report.previous))}
        periodDays={periodDays}
        comparison={comparisonFrom(report)}
        showMedianCycleTime={false}
      />
      <DeliveryCard
        points={report.points}
        breakdown={breakdown}
        onBreakdownChange={setBreakdown}
        intakeSeries={report.intakeSeries}
        hasOutput={report.totals.merged > 0 || report.totals.waste > 0}
        includePeople={report.hasPeopleCohort}
      />
      {report.automations.length > 0 ? (
        <VelocityAutomationsTable
          automations={report.automations}
          organizationId=""
          factoryKey=""
          periodLabel={periodLabel}
          linkNames={false}
        />
      ) : null}
      <CostCard totals={report.totals} points={report.points} />
      <TaskCostCard points={report.points} />
    </>
  );
}

function selectedFactory(factories: AdminVelocityFactory[], factoryId?: string): AdminVelocityFactory | undefined {
  return factories.find((factory) => factory.id === factoryId) ?? factories[0];
}

function summaryCaption(periodLabel: string, periodDays: VelocityPeriodDays, hasPrevious: boolean): string {
  if (hasPrevious) {
    return `${periodLabel}. Compared with the previous ${periodDays} days.`;
  }
  return `${periodLabel}. There is no earlier period to compare with yet.`;
}

function comparisonFrom(report: VelocityReport): VelocityComparison {
  const previous = report.previous;
  if (!previous) {
    return {};
  }
  return {
    tasksClosed: report.totals.tasksClosed - previous.tasksClosed,
    taskWasteRate: report.totals.taskWasteRate - previous.taskWasteRate,
    costPerTask: Math.round((report.totals.costPerTask - previous.costPerTask) * 100) / 100,
  };
}
