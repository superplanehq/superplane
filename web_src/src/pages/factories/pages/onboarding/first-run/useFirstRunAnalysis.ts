import { useBacklogAnalysisScoredOrderIds } from "@/hooks/useBacklogAnalysisRuns";
import { useFactoryIntakeRuns, useFactoryIntakes } from "@/hooks/useFactoryIntakeData";

import { lineIntakeSourceForApiSource } from "../../lineIntakeModel";
import {
  firstRunAnalysisProgress,
  githubIssuesIntake,
  type FirstRunAnalysisProgress,
} from "./firstRunAnalysisProgress";

/**
 * Intakes created before initial-import status existed need the old age
 * fallback. New intakes report their exact import result through the API.
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
  const intake = githubIssuesIntake(intakes.data);
  const runs = useFactoryIntakeRuns(organizationId, factoryId, intake?.id);
  const scoredOrderIds = useBacklogAnalysisScoredOrderIds(organizationId, factoryId);
  const initialImportStatus = intake?.initialImportStatus;
  const githubIntakeMissing = intakes.isSuccess && !intake;
  return {
    progress: firstRunAnalysisProgress(runs.data, scoredOrderIds, importSettled(intake?.createdAt), {
      status: initialImportStatus,
      itemCount: intake?.initialImportItemCount,
    }),
    sourceName: lineIntakeSourceForApiSource(intake?.source)?.name,
    failed: Boolean(
      intakes.isError ||
        runs.isError ||
        githubIntakeMissing ||
        initialImportStatus === "INITIAL_IMPORT_STATUS_FAILED" ||
        initialImportStatus === "INITIAL_IMPORT_STATUS_SKIPPED",
    ),
  };
}
