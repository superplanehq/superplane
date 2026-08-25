import type { FactoriesWorkOrderArtifact } from "@/api-client";

import { OPEN_WORK_ORDER_ARTIFACTS } from "../../__fixtures__/factoryPageFixtureVariants";
import { OPEN_WORK_ORDER_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import type { PhaseGlyphKind } from "../../lib/linePhaseRuns";
import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";

/** Line phase. One line per workspace, so the overlay does not list lines. */
export type RunOverlayPhaseId = "plan" | "implement" | "verify";

export type RunOverlayStepStatus = "passed" | "running" | "failed" | "pending";

export type RunOverlayProvider = "github" | "slack" | "superplane";

export interface RunOverlayStep {
  id: string;
  title: string;
  componentName: string;
  provider: RunOverlayProvider;
  status: RunOverlayStepStatus;
  detail?: string;
  duration?: string;
}

export interface RunOverlayPhase {
  id: RunOverlayPhaseId;
  name: string;
  status: RunOverlayStepStatus;
  duration?: string;
  /** One-line status of this phase. */
  summary: string;
  steps: RunOverlayStep[];
  checkIds: string[];
  artifactIds: string[];
}

export interface RunOverlayFixture {
  title: string;
  /** Short run id. Not a ticket key. */
  runLabel: string;
  elapsed: string;
  currentPhaseId: RunOverlayPhaseId;
  phases: RunOverlayPhase[];
  checks: WorkOrderCheckPresentation[];
  artifacts: FactoriesWorkOrderArtifact[];
}

const CHECKS = presentWorkOrderChecks(OPEN_WORK_ORDER_CHECKS);

/**
 * Mid-flight run on the Implement phase. Plan already attached a note.
 * Implement attached a branch and a pull request, plus the check cards.
 */
export const IMPLEMENT_RUN_OVERLAY: RunOverlayFixture = {
  title: "Ship idempotent refund retries",
  runLabel: "Run 4182",
  elapsed: "12 min",
  currentPhaseId: "implement",
  checks: CHECKS,
  artifacts: OPEN_WORK_ORDER_ARTIFACTS,
  phases: [
    {
      id: "plan",
      name: "Plan",
      status: "passed",
      duration: "4 min",
      summary: "The agent wrote the plan and attached investigation notes.",
      checkIds: ["check-confidence"],
      artifactIds: ["art-md-1"],
      steps: [
        {
          id: "scan-repo",
          title: "Scan repository",
          componentName: "Scan Repository",
          provider: "github",
          status: "passed",
          detail: "ledger · main",
          duration: "18s",
        },
        {
          id: "write-plan",
          title: "Write plan",
          componentName: "Agent",
          provider: "superplane",
          status: "passed",
          detail: "plan.md · 12 steps",
          duration: "1m 16s",
        },
        {
          id: "attach-notes",
          title: "Attach notes",
          componentName: "Add Artifact",
          provider: "superplane",
          status: "passed",
          detail: "Investigation notes",
          duration: "2s",
        },
      ],
    },
    {
      id: "implement",
      name: "Implement",
      status: "running",
      duration: "8 min",
      summary: "The agent is editing the refund dispatcher. Checks update as steps finish.",
      checkIds: ["check-risk-review", "check-code-coverage", "check-test-coverage", "check-ci"],
      artifactIds: ["art-branch-1", "art-pr-1"],
      steps: [
        {
          id: "create-branch",
          title: "Create branch",
          componentName: "Create Branch",
          provider: "github",
          status: "passed",
          detail: "feature/refund-retry",
          duration: "7s",
        },
        {
          id: "open-draft-pr",
          title: "Open draft PR",
          componentName: "Create Pull Request",
          provider: "github",
          status: "passed",
          detail: "PR #482 · Draft",
          duration: "5s",
        },
        {
          id: "implementation",
          title: "Implementation",
          componentName: "Agent",
          provider: "superplane",
          status: "running",
          detail: "RefundDispatcher · 4 files",
          duration: "9m so far",
        },
        {
          id: "coverage-report",
          title: "Coverage report",
          componentName: "Coverage Report",
          provider: "superplane",
          status: "passed",
          detail: "82% overall",
          duration: "41s",
        },
        {
          id: "ci-loop",
          title: "CI loop",
          componentName: "CI Loop",
          provider: "superplane",
          status: "passed",
          detail: "Semaphore #4182",
          duration: "3m 12s",
        },
      ],
    },
    {
      id: "verify",
      name: "Verify",
      status: "pending",
      summary: "Verify starts after Implement finishes. No checks yet.",
      checkIds: [],
      artifactIds: [],
      steps: [
        {
          id: "risk-gate",
          title: "Risk gate",
          componentName: "PR Risk Review",
          provider: "superplane",
          status: "pending",
          detail: "Waits on Implement",
        },
        {
          id: "e2e-refunds",
          title: "Refund E2E",
          componentName: "Run Tests",
          provider: "superplane",
          status: "pending",
          detail: "Waits on Implement",
        },
        {
          id: "merge-ready",
          title: "Mark merge ready",
          componentName: "Update Pull Request",
          provider: "github",
          status: "pending",
          detail: "PR #482",
        },
      ],
    },
  ],
};

export function phaseById(fixture: RunOverlayFixture, id: RunOverlayPhaseId): RunOverlayPhase {
  const phase = fixture.phases.find((entry) => entry.id === id);
  if (!phase) {
    throw new Error(`Unknown run overlay phase: ${id}`);
  }
  return phase;
}

export function checksForPhase(fixture: RunOverlayFixture, phase: RunOverlayPhase): WorkOrderCheckPresentation[] {
  const ids = new Set(phase.checkIds);
  return fixture.checks.filter((check) => ids.has(check.id));
}

export function artifactsForPhase(fixture: RunOverlayFixture, phase: RunOverlayPhase): FactoriesWorkOrderArtifact[] {
  const ids = new Set(phase.artifactIds);
  return fixture.artifacts.filter((artifact) => artifact.id && ids.has(artifact.id));
}

export function nextPhaseId(id: RunOverlayPhaseId): RunOverlayPhaseId | null {
  if (id === "plan") return "implement";
  if (id === "implement") return "verify";
  return null;
}

export function continueActionLabel(id: RunOverlayPhaseId): string {
  if (id === "plan") return "Continue to Implement";
  if (id === "implement") return "Continue to Verify";
  return "Complete run";
}

export function overlayStatusLabel(status: RunOverlayStepStatus): string {
  if (status === "passed") return "Passed";
  if (status === "running") return "Running";
  if (status === "failed") return "Failed";
  return "Pending";
}

export function overlayStatusGlyph(status: RunOverlayStepStatus): PhaseGlyphKind {
  if (status === "running") return "running";
  if (status === "failed") return "failed";
  if (status === "passed") return "passed";
  return "pending";
}
