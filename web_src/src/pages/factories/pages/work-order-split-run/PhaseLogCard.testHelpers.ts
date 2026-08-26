import type { SplitRunPhase, SplitRunStreamLine } from "./splitRunMocks";

/**
 * Shared fixtures for the `PhaseLogCard` spec files so each spec covers one
 * behaviour area with the same phase and stream shape.
 */
export const PHASE: SplitRunPhase = {
  id: "plan",
  name: "Plan",
  status: "passed",
  duration: "1m",
  componentName: "Planning",
  artifacts: [],
  stream: [],
  canvasSteps: [],
};

export function line(
  partial: Partial<SplitRunStreamLine> & Pick<SplitRunStreamLine, "id" | "componentName">,
): SplitRunStreamLine {
  return {
    at: "12:00",
    status: "passed",
    nodeId: "planner-agent",
    ...partial,
  };
}

/** Live log stream state for nodes that stream nothing. */
export function idleLiveLogStream(toggleSection: () => void) {
  return {
    sections: [],
    orphanLines: [],
    error: null,
    isStreaming: false,
    toggleSection,
    scrollRef: { current: null },
  };
}
