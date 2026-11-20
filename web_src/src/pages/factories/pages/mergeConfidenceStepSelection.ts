import { MERGE_CONFIDENCE_CHECKS, type MergeConfidenceCheck } from "./mergeConfidenceChecks";
import { MERGE_CONFIDENCE_CHECK_COPY } from "./mergeConfidenceCopy";
import { mergeCheckId } from "./mergeConfidenceSteps";
import type { PlanningReviewStep } from "./planningReviewMockup";

const NO_CHECKS_STEP: PlanningReviewStep = {
  name: "No checks enabled",
  type: "prompt",
  prompt:
    "Enabled checks: none.\nNo merge confidence checks are enabled. Do not inspect the repository or call the report_merge_check tool.",
};

function builtInCheckId(step: PlanningReviewStep): MergeConfidenceCheck | undefined {
  const id = mergeCheckId(step.prompt);
  // Old and new prompts share the same switch. Keep each saved prompt's scoring scale.
  if (id === "correctness") return "drift";
  return MERGE_CONFIDENCE_CHECKS.find((check) => check === id);
}

function isPlaceholder(step: PlanningReviewStep): boolean {
  return (
    step.name === NO_CHECKS_STEP.name && step.type === NO_CHECKS_STEP.type && step.prompt === NO_CHECKS_STEP.prompt
  );
}

export function enabledMergeChecks(steps: readonly PlanningReviewStep[]): MergeConfidenceCheck[] {
  const ids = new Set(steps.map(builtInCheckId));
  return MERGE_CONFIDENCE_CHECKS.filter((id) => ids.has(id));
}

/** Replace known check slots in canonical order, without moving unrelated steps into the check group. */
export function stepsWithMergeChecks(
  steps: readonly PlanningReviewStep[],
  enabled: readonly MergeConfidenceCheck[],
  defaults: readonly PlanningReviewStep[],
): PlanningReviewStep[] {
  const selected = MERGE_CONFIDENCE_CHECKS.filter((id) => enabled.includes(id)).map((id) => {
    const step =
      steps.find((step) => builtInCheckId(step) === id) ?? defaults.find((step) => builtInCheckId(step) === id);
    if (step) return step;
    const label = MERGE_CONFIDENCE_CHECK_COPY.find((check) => check.id === id)?.label ?? id;
    throw new Error(`Could not load the default ${label} check. Try saving again.`);
  });
  const checkSlots = steps.flatMap((step, index) => (builtInCheckId(step) || isPlaceholder(step) ? [index] : []));
  const lastCheckSlot = checkSlots[checkSlots.length - 1] ?? -1;
  const result: PlanningReviewStep[] = [];
  let nextCheck = 0;
  steps.forEach((step, index) => {
    if (!builtInCheckId(step) && !isPlaceholder(step)) {
      result.push(step);
      return;
    }
    if (nextCheck < selected.length) result.push(selected[nextCheck++]);
    if (index === lastCheckSlot) result.push(...selected.slice(nextCheck));
  });
  if (lastCheckSlot < 0) result.push(...selected);
  if (!result.some((step) => mergeCheckId(step.prompt) !== null)) result.push({ ...NO_CHECKS_STEP });
  return result;
}
