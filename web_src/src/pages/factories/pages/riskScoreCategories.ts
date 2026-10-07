import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

export type RiskScoreLevel = 1 | 2 | 3;

export type RiskScoreCategory = {
  id: string;
  name: string;
  score: RiskScoreLevel;
};

/** Default category floors. The pull request score is the highest match. */
export const RISK_SCORE_CATEGORIES: RiskScoreCategory[] = [
  { id: "documentation", name: "Documentation only", score: 1 },
  { id: "tests", name: "Tests only", score: 1 },
  { id: "interface", name: "User interface changes", score: 1 },
  { id: "database-additive", name: "Additive database changes", score: 2 },
  { id: "dependencies", name: "Dependency updates", score: 2 },
  { id: "api", name: "API behavior changes", score: 2 },
  { id: "authorization", name: "Authorization changes", score: 3 },
  { id: "authentication", name: "Authentication changes", score: 3 },
  { id: "data-migration", name: "Data deletion or migration", score: 3 },
  { id: "infrastructure", name: "Infrastructure changes", score: 3 },
  { id: "billing", name: "Billing and payment changes", score: 3 },
  { id: "secrets", name: "Secrets and credentials", score: 3 },
];

const LEVEL_WORD: Record<RiskScoreLevel, string> = {
  1: "healthy",
  2: "caution",
  3: "critical",
};

export function defaultRiskScoreCategories(): RiskScoreCategory[] {
  return RISK_SCORE_CATEGORIES.map((category) => ({ ...category }));
}

/** A category name must stay one rules token. An equals sign breaks the saved line. */
export function isRiskScoreCategoryName(name: string): boolean {
  const trimmed = name.trim();
  return trimmed.length > 0 && !trimmed.includes("=") && !trimmed.includes("\n");
}

export function formatRiskScoreRules(categories: Array<{ name: string; score: number }>): string {
  return categories
    .map((category) => {
      const name = category.name.trim();
      if (!isRiskScoreCategoryName(name)) {
        return "";
      }
      const score = clampRiskScoreLevel(category.score);
      return `${name} = ${score} (${LEVEL_WORD[score]}).`;
    })
    .filter((rule) => rule.length > 0)
    .join(" ");
}

export function nextRiskScoreCategoryId(categories: RiskScoreCategory[]): string {
  const used = categories
    .map((category) => /^custom-(\d+)$/.exec(category.id)?.[1])
    .filter((value): value is string => Boolean(value))
    .map(Number);
  return `custom-${Math.max(0, ...used) + 1}`;
}

const LEVEL_WORDS = "very_low|low|medium|high|critical|healthy|caution";
const RULE_PATTERN = new RegExp(`([^=\\n]+?) = ([1-5]) \\((${LEVEL_WORDS})\\)\\.`, "g");
const RULES_LINE_PATTERN = new RegExp(`^(?:[^=\\n]+? = [1-5] \\((?:${LEVEL_WORDS})\\)\\.\\s*)+$`, "m");

/** Reads the category rules line from an agent prompt. Returns null when the prompt has no rules line. */
export function parseRiskScoreRules(prompt: string): RiskScoreCategory[] | null {
  const line = RULES_LINE_PATTERN.exec(prompt)?.[0];
  if (!line) {
    return null;
  }
  const categories: RiskScoreCategory[] = [];
  for (const match of line.matchAll(RULE_PATTERN)) {
    const name = match[1].trim();
    const known = RISK_SCORE_CATEGORIES.find((category) => category.name === name);
    categories.push({
      id: known?.id ?? nextRiskScoreCategoryId(categories),
      name,
      score: scoreFromRule(Number(match[2]), match[3]),
    });
  }
  return categories;
}

/** Replaces the category rules line in an agent prompt. Returns null when the prompt has no rules line. */
export function replaceRiskScoreRules(prompt: string, categories: RiskScoreCategory[]): string | null {
  if (!RULES_LINE_PATTERN.test(prompt)) {
    return null;
  }
  return prompt.replace(RULES_LINE_PATTERN, formatRiskScoreRules(categories));
}

export function riskScoreCategoriesFromDraft(draft: PlanningReviewDraft | undefined): RiskScoreCategory[] | null {
  for (const step of draftSteps(draft)) {
    const categories = step.prompt ? parseRiskScoreRules(step.prompt) : null;
    if (categories) {
      return categories;
    }
  }
  return null;
}

/** Returns the draft with new rules in the first prompt step that has a rules line. */
export function draftWithRiskScoreCategories(
  draft: PlanningReviewDraft,
  categories: RiskScoreCategory[],
): PlanningReviewDraft | null {
  const [component, ...rest] = draft.components;
  if (!component) {
    return null;
  }
  const steps = draftSteps(draft);
  const index = steps.findIndex((step) => step.prompt && parseRiskScoreRules(step.prompt));
  const prompt = index >= 0 ? replaceRiskScoreRules(steps[index].prompt ?? "", categories) : null;
  if (prompt === null) {
    return null;
  }
  const nextSteps = steps.map((step, stepIndex) => {
    const source = stepIndex === index ? prompt : (step.prompt ?? "");
    if (!source) {
      return step;
    }
    return { ...step, prompt: withCurrentMergeCheckScale(source) };
  });
  return {
    ...draft,
    components: [{ ...component, configuration: { ...component.configuration, steps: nextSteps } }, ...rest],
  };
}

function draftSteps(draft: PlanningReviewDraft | undefined): PlanningReviewStep[] {
  const steps = draft?.components[0]?.configuration.steps;
  return Array.isArray(steps) ? (steps as PlanningReviewStep[]) : [];
}

const LEGACY_MERGE_CHECK_SCORE =
  /(?:\b1 to 5\b|\b1 through 5\b|report(?: the check)? with 5\b|(?:^|\n)\s*[45] means\b| = [45] \()/;

const MERGE_CHECK_SCALE_REPLACEMENTS: ReadonlyArray<readonly [string, string]> = [
  [
    "Use Higher risk for high or critical. Use Moderate risk for medium. Use Lower risk for low or very low.",
    "Use Higher risk for critical. Use Moderate risk for caution. Use Lower risk for healthy.",
  ],
  [
    [
      "5 means the change follows every practice that applies.",
      "4 means the change follows the practices that apply, with a small gap.",
      "3 means the change follows some practices and breaks one that applies.",
      "2 means the change breaks a practice that applies.",
      "1 means the change breaks more than one practice that applies.",
      "If no performance practice applies, report the check with 5.",
    ].join("\n"),
    [
      "3 means the change follows the practices that apply, including a small gap.",
      "2 means the change follows some practices and breaks one that applies.",
      "1 means the change breaks one or more practices that apply.",
      "If no performance practice applies, report 3.",
    ].join("\n"),
  ],
  ["If no security practice applies, report the check with 5.", "If no security practice applies, report 3."],
  [
    [
      "1 means the change does what the task asks.",
      "2 means the change does the task and also makes a small extra change.",
      "3 means the change does the task and also adds work that raises risk past the task.",
      "4 means the change does most of the task and also does something the task did not ask for.",
      "5 means the change does something different from the task, or raises risk far past the task.",
    ].join("\n"),
    [
      "1 means the change does what the task asks, including a small extra change.",
      "2 means the change does the task and also adds work that raises risk past the task.",
      "3 means the change does something the task did not ask for, or raises risk far past the task.",
    ].join("\n"),
  ],
  [
    [
      "5 means a revert undoes the change and no data is lost.",
      "4 means a revert undoes the change after one small manual step.",
      "3 means a person must repair data or configuration to undo the change.",
      "2 means part of the change stays after a revert.",
      "1 means you cannot undo the change without data loss or a new change.",
    ].join("\n"),
    [
      "3 means a revert undoes the change, including after one small manual step, and no data is lost.",
      "2 means a person must repair data or configuration to undo the change.",
      "1 means part of the change stays after a revert, or the change cannot be undone without data loss or a new change.",
    ].join("\n"),
  ],
  ["score is an integer from 1 to 5", "score is an integer from 1 to 3"],
  ["choose a score from 1 to 5", "choose a score from 1 to 3"],
  ["1 through 5", "1 through 3"],
];

/** Rewrites a stored 1 to 5 prompt only when the result no longer asks for 5. */
function withCurrentMergeCheckScale(prompt: string): string {
  let next = prompt;
  for (const [from, to] of MERGE_CHECK_SCALE_REPLACEMENTS) {
    next = next.split(from).join(to);
  }
  if (LEGACY_MERGE_CHECK_SCORE.test(next)) {
    return prompt;
  }
  return next;
}

function scoreFromRule(raw: number, word: string): RiskScoreLevel {
  if (word === "healthy" || word === "caution" || (word === "critical" && raw <= 3)) {
    return clampRiskScoreLevel(raw);
  }
  return foldLegacyRiskScore(raw);
}

function foldLegacyRiskScore(value: number): RiskScoreLevel {
  if (value <= 2) {
    return 1;
  }
  if (value === 3) {
    return 2;
  }
  return 3;
}

function clampRiskScoreLevel(value: number): RiskScoreLevel {
  if (value <= 1) {
    return 1;
  }
  if (value >= 3) {
    return 3;
  }
  return 2;
}
