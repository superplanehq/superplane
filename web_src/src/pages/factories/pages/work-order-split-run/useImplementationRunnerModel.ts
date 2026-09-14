import type { SplitRunPhase } from "./splitRunMocks";
import { implementationRunnerModel, selectedImplementationPhase } from "./splitRunWorkOrderDisplay";
import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";

export function useImplementationRunnerModel(organizationId: string | undefined, phases: SplitRunPhase[]): string {
  const selected = selectedImplementationPhase(phases);
  const live = useSplitRunLiveCanvas(organizationId, selected);
  return implementationRunnerModel(phases, live.canvas?.nodes);
}
