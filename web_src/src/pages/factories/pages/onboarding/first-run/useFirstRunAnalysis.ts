import { useBacklogAnalysisScoredOrderIds } from "@/hooks/useBacklogAnalysisRuns";
import { useFactoryIntakeRuns, useFactoryIntakes } from "@/hooks/useFactoryIntakeData";

import { lineIntakeSourceForApiSource } from "../../lineIntakeModel";
import { firstRunAnalysisProgress, type FirstRunAnalysisProgress } from "./firstRunAnalysisProgress";

/**
 * The API has no "import finished" flag. The seed import starts when
 * provisioning creates the intake, so an intake this old with zero runs
 * means the ticket source had no open tickets. The runs query polls every
 * ten seconds, so the check re-evaluates without its own timer.
 */
const IMPORT_GRACE_MS = 30_000;

function importSettled(createdAt?: string): boolean {
  if (!createdAt) return false;
  return Date.now() - new Date(createdAt).getTime() >= IMPORT_GRACE_MS;
}

/**
 * Progress of the intake scoring that provisioning started. Polling uses the
 * organization id returned by provisioning, because the initial path may have
 * renamed the organization slug. Scoring itself can run in the factory
 * Backlog automation, so its finished runs also count as scored.
 */
export function useFirstRunAnalysis(
  organizationId: string,
  factoryId: string,
): { progress: FirstRunAnalysisProgress; sourceName?: string; failed: boolean } {
  const intakes = useFactoryIntakes(organizationId, factoryId);
  const intake = intakes.data?.[0];
  const runs = useFactoryIntakeRuns(organizationId, factoryId, intake?.id);
  const scoredOrderIds = useBacklogAnalysisScoredOrderIds(organizationId, factoryId);
  return {
    progress: firstRunAnalysisProgress(runs.data, scoredOrderIds, importSettled(intake?.createdAt)),
    sourceName: lineIntakeSourceForApiSource(intake?.source)?.name,
    failed: Boolean(intakes.isError || runs.isError),
  };
}
