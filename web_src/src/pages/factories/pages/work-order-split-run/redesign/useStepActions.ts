import { useMemo } from "react";

import { useWorkOrderEvents } from "@/hooks/useFactoryData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { FEATURE_FACTORY_STEP_ACTIONS } from "@/lib/experimentalFeatures";
import { flattenWorkOrderEventsPages } from "../../../lib/workOrderEventsPagination";
import type { ConsoleAutomation } from "./automationsViewModel";
import { stepActionRows, type StepActionRow } from "./consoleStepActions";
import type { SplitRunPhase } from "../splitRunMocks";

export function useStepActions(
  organizationId: string,
  factoryId: string,
  orderId: string,
  automation: ConsoleAutomation,
  phases: SplitRunPhase[],
): StepActionRow[] {
  const { has } = useExperimentalFeature(organizationId);
  const events = useWorkOrderEvents(organizationId, factoryId, orderId);
  const { resolveUser } = useOrgUserLookup(organizationId);
  const enabled = has(FEATURE_FACTORY_STEP_ACTIONS);

  return useMemo(() => {
    if (!enabled) {
      return [];
    }
    return stepActionRows(
      flattenWorkOrderEventsPages(events.data?.pages),
      automationRunIds(automation, phases),
      (userId) => resolveUser(userId)?.name,
    );
  }, [automation, enabled, events.data?.pages, phases, resolveUser]);
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
