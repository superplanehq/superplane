import { useState } from "react";

import { cn } from "@/lib/utils";
import { SegmentedNav } from "@/ui/SegmentedNav";

import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { VELOCITY_PERIOD_OPTIONS } from "../lib/factoryVelocityFlow";
import { FACTORY_VELOCITY_FLOW_BY_PERIOD } from "./factoryVelocityFlowMockData";
import {
  FACTORY_VELOCITY_BY_PERIOD,
  FACTORY_VELOCITY_YESTERDAY,
  type FactoryVelocityDay,
  type FactoryVelocityPeriodDays,
} from "./factoryVelocityMockData";
import { factorySectionBodyClassName, factorySectionHeaderClassName } from "./factoryPageLayoutStyles";
import {
  VelocityLoadedView,
  type VelocityCostConfig,
  type VelocityData,
  type VelocitySourceSplitConfig,
  type VelocityWorkOrderFlowConfig,
} from "./VelocityLoadedView";

function pointsFromMock(points: FactoryVelocityDay[]) {
  return points.map((point) => ({
    day: point.day,
    merged: point.merged,
    waste: point.waste,
    peopleMerged: point.peopleMerged,
    superplaneMerged: point.superplaneMerged,
  }));
}

function toVelocityDataFromMock(periodDays: FactoryVelocityPeriodDays): VelocityData {
  const period = FACTORY_VELOCITY_BY_PERIOD[periodDays];
  return {
    yesterday: {
      dateLabel: FACTORY_VELOCITY_YESTERDAY.dateLabel,
      merged: FACTORY_VELOCITY_YESTERDAY.merged,
      waste: FACTORY_VELOCITY_YESTERDAY.waste,
      wastePct: FACTORY_VELOCITY_YESTERDAY.wastePct,
    },
    totals: {
      merged: period.totals.merged,
      waste: period.totals.waste,
      wastePct: period.totals.wastePct,
      superplaneMerged: period.totals.superplaneMerged,
      peopleMerged: period.totals.peopleMerged,
      superplaneSharePct: period.totals.superplaneSharePct,
    },
    points: pointsFromMock(period.points),
  };
}

function toCostConfigFromMock(periodDays: FactoryVelocityPeriodDays): VelocityCostConfig {
  const period = FACTORY_VELOCITY_BY_PERIOD[periodDays];
  return {
    yesterdayCostUsd: FACTORY_VELOCITY_YESTERDAY.costUsd,
    yesterdayTokens: FACTORY_VELOCITY_YESTERDAY.tokens,
    yesterdayCostPerMerged: FACTORY_VELOCITY_YESTERDAY.costPerMergedPr,
    totalCostUsd: period.totals.costUsd,
    seriesUsd: period.points.map((point) => point.costUsd),
  };
}

function toWorkOrderFlowFromMock(periodDays: FactoryVelocityPeriodDays): VelocityWorkOrderFlowConfig {
  const mock = FACTORY_VELOCITY_FLOW_BY_PERIOD[periodDays];
  return {
    flow: {
      days: periodDays,
      label: mock.label,
      sampleSize: 42,
      medianCycleHours: mock.medianCycleHours,
      medianRunningHours: mock.medianRunningHours,
      medianWaitingHours: mock.medianWaitingHours,
      runningShareOfCyclePct: mock.runningShareOfCyclePct,
      waitingShareOfCyclePct: mock.waitingShareOfCyclePct,
      timeTrend: mock.timeTrend,
    },
  };
}

export function VelocityPrototypePage() {
  const [periodDays, setPeriodDays] = useState<FactoryVelocityPeriodDays>(7);
  const period = FACTORY_VELOCITY_BY_PERIOD[periodDays];

  const sourceSplit: VelocitySourceSplitConfig = {
    hasPeopleCohort: true,
    repositoryLabel: "acme/refunds",
  };

  return (
    <>
      <WorkspacePageHeader
        className={factorySectionHeaderClassName}
        title="Velocity"
        subtitle="Merged pull requests, waste, cost, and work order time."
        actions={
          <SegmentedNav
            ariaLabel="Velocity period in days"
            size="xs"
            value={String(periodDays)}
            onValueChange={(value) => {
              const next = Number(value);
              if (next === 7 || next === 30) setPeriodDays(next);
            }}
            options={VELOCITY_PERIOD_OPTIONS}
          />
        }
      />

      <div className={cn(factorySectionBodyClassName, "space-y-6")} data-testid="factory-velocity-page">
        <VelocityLoadedView
          periodLabel={period.label}
          periodDays={periodDays}
          data={toVelocityDataFromMock(periodDays)}
          sourceSplit={sourceSplit}
          workOrderFlow={toWorkOrderFlowFromMock(periodDays)}
          cost={toCostConfigFromMock(periodDays)}
        />
      </div>
    </>
  );
}
