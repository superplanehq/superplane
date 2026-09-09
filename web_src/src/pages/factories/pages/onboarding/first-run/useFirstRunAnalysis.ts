import { useBacklogAnalysisScoredOrderIds } from "@/hooks/useBacklogAnalysisRuns";
import { useFactoryIntakeRuns, useFactoryIntakes } from "@/hooks/useFactoryIntakeData";

import { firstRunAnalysisProgress, type FirstRunAnalysisProgress } from "./firstRunAnalysisProgress";

/**
 * Progress of the intake scoring that provisioning started. Polling uses the
 * organization id returned by provisioning, because the initial path may have
 * renamed the organization slug. Scoring itself can run in the factory
 * Backlog automation, so its finished runs also count as scored.
 */
export function useFirstRunAnalysis(
  organizationId: string,
  factoryId: string,
): { progress: FirstRunAnalysisProgress; failed: boolean } {
  const intakes = useFactoryIntakes(organizationId, factoryId);
  const intakeId = intakes.data?.[0]?.id;
  const runs = useFactoryIntakeRuns(organizationId, factoryId, intakeId);
  const scoredOrderIds = useBacklogAnalysisScoredOrderIds(organizationId, factoryId);
  return {
    progress: firstRunAnalysisProgress(runs.data, scoredOrderIds),
    failed: Boolean(intakes.isError || runs.isError),
  };
}
