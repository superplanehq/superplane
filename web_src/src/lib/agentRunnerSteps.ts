import { expandMergeConfidenceSteps } from "@/pages/factories/pages/mergeConfidenceSteps";
import type { PlanningReviewStep } from "@/pages/factories/pages/planningReviewMockup";

/** Titles of configured bash and prompt steps, in execution order. */
export function agentRunnerStepTitles(configuration: unknown, options?: { expandMergeChecks?: boolean }): string[] {
  const steps = configuredSteps(configuration);
  if (!steps) {
    return [];
  }
  const listed = options?.expandMergeChecks ? expandMergeConfidenceSteps(steps) : steps;

  return listed.map((step) => step.name.trim()).filter((name) => name.length > 0);
}

function configuredSteps(configuration: unknown): PlanningReviewStep[] | null {
  if (!configuration || typeof configuration !== "object") {
    return null;
  }
  const steps = (configuration as Record<string, unknown>).steps;
  if (!Array.isArray(steps)) {
    return null;
  }
  return steps.flatMap(planningStep);
}

function planningStep(step: unknown): PlanningReviewStep[] {
  if (!step || typeof step !== "object") {
    return [];
  }
  const record = step as Record<string, unknown>;
  const name = typeof record.name === "string" ? record.name : "";
  return [
    {
      name,
      type: record.type === "bash" ? "bash" : "prompt",
      prompt: typeof record.prompt === "string" ? record.prompt : undefined,
      workingDirectory: typeof record.workingDirectory === "string" ? record.workingDirectory : undefined,
    },
  ];
}

export const AGENT_HARNESS_COMPONENTS = new Set<string>([
  "runnerSuperPlane",
  "runnerClaudeCode",
  "runnerCodex",
  "runnerOpenRouter",
]);

export function isAgentHarnessComponent(component: string | undefined): boolean {
  return Boolean(component && AGENT_HARNESS_COMPONENTS.has(component));
}
