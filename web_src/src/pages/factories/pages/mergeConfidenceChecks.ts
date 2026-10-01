import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";
import { draftWithRiskScoreCategories, parseRiskScoreRules, type RiskScoreCategory } from "./riskScoreCategories";

export const MERGE_CONFIDENCE_CHECKS = ["risk", "performance", "security", "drift"] as const;

export type MergeConfidenceCheck = (typeof MERGE_CONFIDENCE_CHECKS)[number];

export type MergeConfidenceSettings = {
  checks: MergeConfidenceCheck[];
  categories: RiskScoreCategory[];
};

const ENABLED_CHECKS_LINE =
  /^Enabled checks: (none|(?:risk|performance|security|drift)(?:, (?:risk|performance|security|drift))*)\.$/m;

export function defaultMergeConfidenceChecks(): MergeConfidenceCheck[] {
  return [...MERGE_CONFIDENCE_CHECKS];
}

/** The install-param value. `none` means every check is off. */
export function formatEnabledChecksValue(checks: readonly MergeConfidenceCheck[]): string {
  const ordered = MERGE_CONFIDENCE_CHECKS.filter((check) => checks.includes(check));
  return ordered.length === 0 ? "none" : ordered.join(", ");
}

export function formatEnabledChecksLine(checks: readonly MergeConfidenceCheck[]): string {
  return `Enabled checks: ${formatEnabledChecksValue(checks)}.`;
}

/** Reads the enabled-checks line from an agent prompt. Returns null when the line is missing or invalid. */
export function parseEnabledChecks(prompt: string): MergeConfidenceCheck[] | null {
  const match = ENABLED_CHECKS_LINE.exec(prompt);
  if (!match) {
    return null;
  }
  if (match[1] === "none") {
    return [];
  }
  const names = match[1].split(", ").filter((name): name is MergeConfidenceCheck => isMergeConfidenceCheck(name));
  if (names.length !== match[1].split(", ").length || new Set(names).size !== names.length) {
    return null;
  }
  return MERGE_CONFIDENCE_CHECKS.filter((check) => names.includes(check));
}

export function replaceEnabledChecks(prompt: string, checks: readonly MergeConfidenceCheck[]): string | null {
  if (!ENABLED_CHECKS_LINE.test(prompt)) {
    return null;
  }
  return prompt.replace(ENABLED_CHECKS_LINE, formatEnabledChecksLine(checks));
}

export function mergeConfidenceFromDraft(draft: PlanningReviewDraft | undefined): MergeConfidenceSettings | null {
  for (const step of draftSteps(draft)) {
    if (!step.prompt) {
      continue;
    }
    const checks = parseEnabledChecks(step.prompt);
    const categories = parseRiskScoreRules(step.prompt);
    if (checks && categories) {
      return { checks, categories };
    }
  }
  return null;
}

/** Returns the draft with a new checks line and risk rules in the review prompt. */
export function draftWithMergeConfidence(
  draft: PlanningReviewDraft,
  settings: MergeConfidenceSettings,
): PlanningReviewDraft | null {
  const withRules = draftWithRiskScoreCategories(draft, settings.categories);
  if (!withRules) {
    return null;
  }
  const [component, ...rest] = withRules.components;
  if (!component) {
    return null;
  }
  const steps = draftSteps(withRules);
  const index = steps.findIndex((step) => step.prompt && parseEnabledChecks(step.prompt));
  const prompt = index >= 0 ? replaceEnabledChecks(steps[index].prompt ?? "", settings.checks) : null;
  if (prompt === null) {
    return null;
  }
  const nextSteps = steps.map((step, stepIndex) => (stepIndex === index ? { ...step, prompt } : step));
  return {
    ...withRules,
    components: [{ ...component, configuration: { ...component.configuration, steps: nextSteps } }, ...rest],
  };
}

function draftSteps(draft: PlanningReviewDraft | undefined): PlanningReviewStep[] {
  const steps = draft?.components[0]?.configuration.steps;
  return Array.isArray(steps) ? (steps as PlanningReviewStep[]) : [];
}

function isMergeConfidenceCheck(value: string): value is MergeConfidenceCheck {
  return (MERGE_CONFIDENCE_CHECKS as readonly string[]).includes(value);
}
