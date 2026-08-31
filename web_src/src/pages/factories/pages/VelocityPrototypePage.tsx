import { useMemo, useState } from "react";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { workOrdersPath } from "../lib/factoryPagePaths";
import {
  CostCard,
  DeliveryCard,
  SummaryCard,
  TaskTimeCard,
  VelocityPrototypeChrome,
  type VelocityComparison,
} from "./velocityPrototypeCards";
import { VelocityPeopleTable } from "./VelocityPeopleTable";
import { VelocityZeroState } from "./VelocityZeroState";
import {
  EARLY_USAGE_CLOSED_TASKS,
  buildEarlyUsageVelocityPoints,
  buildPeople,
  buildVelocityPoints,
  summarizePoints,
  type Breakdown,
  type PeriodDays,
  type VelocitySummary,
} from "./velocityPrototypeData";

function usePeopleShare(summary: VelocitySummary) {
  return useMemo(
    () =>
      buildPeople({
        peopleMerged: summary.peopleMerged,
        superplaneMerged: summary.superplaneMerged,
        waste: summary.waste,
        costUsd: summary.cost,
      }),
    [summary.peopleMerged, summary.superplaneMerged, summary.waste, summary.cost],
  );
}

/** First-run empty report: same chrome as the populated prototype, no charts. */
export function VelocityPrototypeZeroStatePage() {
  const [periodDays, setPeriodDays] = useState<PeriodDays>(14);
  const { organizationId, factoryKey } = useFactoriesLayout();

  return (
    <VelocityPrototypeChrome periodDays={periodDays} onPeriodDaysChange={setPeriodDays}>
      <VelocityZeroState tasksHref={workOrdersPath(organizationId, factoryKey)} />
    </VelocityPrototypeChrome>
  );
}

/**
 * A workspace a few hours old. Every card holds real numbers, but SuperPlane
 * has one day of output, so the page drops the period comparison and names the
 * sample behind each median.
 */
export function VelocityPrototypeEarlyUsagePage() {
  const [periodDays, setPeriodDays] = useState<PeriodDays>(14);
  const [breakdown, setBreakdown] = useState<Breakdown>("origin");
  const points = useMemo(() => buildEarlyUsageVelocityPoints(periodDays), [periodDays]);

  const summary = summarizePoints(points);
  const people = usePeopleShare(summary);
  const periodLabel = `Last ${periodDays} days`;
  const sampleNote = `From ${EARLY_USAGE_CLOSED_TASKS} tasks closed today`;

  return (
    <VelocityPrototypeChrome periodDays={periodDays} onPeriodDaysChange={setPeriodDays}>
      <SummaryCard
        summary={summary}
        caption={`${periodLabel}. SuperPlane started today, so there is no comparison yet.`}
      />
      <DeliveryCard points={points} breakdown={breakdown} onBreakdownChange={setBreakdown} />
      <VelocityPeopleTable people={people} periodLabel={periodLabel} />
      <div className="grid gap-5 lg:grid-cols-2">
        <TaskTimeCard summary={summary} points={points} sampleNote={sampleNote} />
        <CostCard summary={summary} points={points} sampleNote={sampleNote} />
      </div>
    </VelocityPrototypeChrome>
  );
}

export function VelocityPrototypePage() {
  const [periodDays, setPeriodDays] = useState<PeriodDays>(14);
  const [breakdown, setBreakdown] = useState<Breakdown>("origin");
  const points = useMemo(() => buildVelocityPoints(periodDays, periodDays), [periodDays]);
  const previousPoints = useMemo(() => buildVelocityPoints(periodDays, 0), [periodDays]);

  const summary = summarizePoints(points);
  const previous = summarizePoints(previousPoints);
  const people = usePeopleShare(summary);
  const periodLabel = `Last ${periodDays} days`;

  const comparison: VelocityComparison = {
    merged: summary.merged - previous.merged,
    wasteRate: summary.wasteRate - previous.wasteRate,
    cycleHours: Math.round(summary.cycleHours - previous.cycleHours),
    costPerMerge: Math.round((summary.costPerMerge - previous.costPerMerge) * 100) / 100,
  };

  return (
    <VelocityPrototypeChrome periodDays={periodDays} onPeriodDaysChange={setPeriodDays}>
      <SummaryCard
        summary={summary}
        caption={`${periodLabel}. Compared with the previous ${periodDays} days.`}
        comparison={comparison}
      />
      <DeliveryCard points={points} breakdown={breakdown} onBreakdownChange={setBreakdown} />
      <VelocityPeopleTable people={people} periodLabel={periodLabel} />
      <div className="grid gap-5 lg:grid-cols-2">
        <TaskTimeCard summary={summary} points={points} />
        <CostCard summary={summary} points={points} />
      </div>
    </VelocityPrototypeChrome>
  );
}
