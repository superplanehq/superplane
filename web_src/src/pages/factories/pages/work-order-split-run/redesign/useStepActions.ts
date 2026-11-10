import { useEffect, useMemo } from "react";

import { useWorkOrderEvents } from "@/hooks/useFactoryData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { FEATURE_FACTORY_STEP_ACTIONS } from "@/lib/experimentalFeatures";
import { flattenWorkOrderEventsPages } from "../../../lib/workOrderEventsPagination";
import { useOptionalFactoriesLayout } from "../../../layout/factoriesLayoutContext";
import { factoryAppRunPath } from "../../../lib/factoryPagePaths";
import { humanReviewActionRows } from "./consoleHumanReviewActions";
import type { ConsoleAutomation } from "./automationsViewModel";
import { stepActionRows, type StepActionRow } from "./consoleStepActions";
import type { SplitRunPhase } from "../splitRunMocks";

export function useStepActions(
  organizationId: string,
  factoryId: string,
  orderId: string,
  automation: ConsoleAutomation,
  phases: SplitRunPhase[],
): { rows: StepActionRow[]; isError: boolean; retry: () => void } {
  const { has } = useExperimentalFeature(organizationId);
  const events = useWorkOrderEvents(organizationId, factoryId, orderId);
  const { data, hasNextPage, isFetchingNextPage, isError, fetchNextPage, refetch } = events;
  const { resolveUser } = useOrgUserLookup(organizationId);
  const enabled = has(FEATURE_FACTORY_STEP_ACTIONS);
  const layout = useOptionalFactoriesLayout();
  const factoryKey = layout?.routeSegment ?? factoryId;

  useEffect(() => {
    if (!enabled || isError || !hasNextPage || isFetchingNextPage) {
      return;
    }
    void fetchNextPage();
  }, [enabled, fetchNextPage, hasNextPage, isError, isFetchingNextPage]);

  const rows = useMemo(() => {
    if (!enabled) {
      return [];
    }
    const runIds = automationRunIds(automation, phases);
    const eventRows = stepActionRows(
      flattenWorkOrderEventsPages(data?.pages),
      runIds,
      (userId) => resolveUser(userId)?.name,
    );
    return [
      ...eventRows,
      ...humanReviewActionRows(phases, runIds, (canvasId, runId) =>
        factoryAppRunPath(organizationId, factoryKey, canvasId, runId),
      ),
    ];
  }, [automation, data?.pages, enabled, phases, resolveUser, organizationId, factoryKey]);

  return {
    rows,
    isError: enabled && isError,
    retry: () => {
      void refetch();
    },
  };
}

function automationRunIds(automation: ConsoleAutomation, phases: SplitRunPhase[]): Set<string> {
  const stageIds = new Set(automation.runs.map((run) => run.id));
  const ids = new Set<string>();
  for (const phase of phases) {
    if (stageIds.has(phase.id) && phase.runId) {
      ids.add(phase.runId);
    }
  }
  return ids;
}
