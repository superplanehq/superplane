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

export const LONG_NOTE =
  "Now let me check factories.proto Delete rpc absence explicitly and PermissionTooltip component quickly, plus check showSuccessToast import paths.";

export const PLANNING_STREAM: SplitRunStreamLine[] = [
  line({ id: "planner-agent", componentName: "Agent - Plan for GH Issue", componentType: "Run Claude Code" }),
  line({
    id: "step-clone",
    note: true,
    componentName: "Clone Repo",
    componentType: "bash",
    status: "passed",
    detail: "Cloning into 'superplane'...",
  }),
  line({
    id: "step-fail",
    note: true,
    componentName: "Run Tests",
    componentType: "bash",
    status: "failed",
    detail: "FAIL pkg/foo",
  }),
  line({
    id: "step-write",
    note: true,
    componentName: "Write Implementation Plan",
    componentType: "prompt",
  }),
  line({
    id: "cmd-cat",
    note: true,
    noteParentId: "step-write",
    noteDepth: 1,
    componentName: "cat /tmp/ORDER.md",
    componentType: "bash",
    detail: "## Goal\nAdd a menu.",
  }),
  line({
    id: "cmd-note",
    note: true,
    noteParentId: "step-write",
    noteDepth: 1,
    componentName: LONG_NOTE,
    componentType: "note",
  }),
  line({
    id: "cmd-read",
    note: true,
    noteParentId: "step-write",
    noteDepth: 1,
    componentName: "LineListCard.tsx",
    componentType: "read",
  }),
  line({ id: "step-out", note: true, componentName: "Use plan as output", componentType: "bash" }),
];

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
