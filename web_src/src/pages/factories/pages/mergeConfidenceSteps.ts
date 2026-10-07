import { parseEnabledChecks, type MergeConfidenceCheck } from "./mergeConfidenceChecks";
import type { PlanningReviewStep } from "./planningReviewMockup";

const MERGE_CHECK_HEADER = /^Merge check: ([a-z][a-z0-9-]{0,40})\.\n?/;

const CHECK_SECTIONS: ReadonlyArray<{ id: MergeConfidenceCheck; label: string; marker: string }> = [
  { id: "risk", label: "Blast radius", marker: "Risk. Use this section only when risk is enabled." },
  {
    id: "performance",
    label: "Performance",
    marker: "Performance. Use this section only when performance is enabled.",
  },
  { id: "security", label: "Security", marker: "Security. Use this section only when security is enabled." },
  { id: "drift", label: "Drift from Specification", marker: "Drift. Use this section only when drift is enabled." },
  {
    id: "reversibility",
    label: "Reversibility",
    marker: "Reversibility. Use this section only when reversibility is enabled.",
  },
];

function newCheckPrompt(id: string): string {
  return [
    "Describe what this check looks for.",
    "The repository is checked out at the pull request head.",
    "The pull request diff is in /tmp/pr.diff.",
    "Report this check with the report_merge_check tool.",
    "Call the tool once.",
    `check is ${id}.`,
    "score is an integer from 1 to 3. A higher score means a worse result.",
    "summary is one sentence.",
    "Do not write a file.",
  ].join("\n");
}

/** A merge confidence agent stores one prompt step per check. */
export function isMergeConfidenceSteps(steps: readonly PlanningReviewStep[]): boolean {
  return steps.some((step) => mergeCheckId(step.prompt) !== null || isCombinedReviewPrompt(step.prompt));
}

/** Shows each enabled check as its own step. A prompt that is already split stays as it is. */
export function expandMergeConfidenceSteps(steps: readonly PlanningReviewStep[]): PlanningReviewStep[] {
  if (steps.some((step) => mergeCheckId(step.prompt) !== null)) {
    return [...steps];
  }
  const index = steps.findIndex((step) => isCombinedReviewPrompt(step.prompt));
  const review = steps[index];
  if (!review?.prompt) {
    return [...steps];
  }
  const checks = checkStepsFromReview(review);
  return [...steps.slice(0, index), ...checks, ...steps.slice(index + 1)];
}

/** The next custom check. The id stays on the step when the name changes. */
export function newMergeConfidenceStep(steps: readonly PlanningReviewStep[]): PlanningReviewStep {
  const id = nextCheckId(steps);
  return {
    name: "New check",
    type: "prompt",
    workingDirectory: "repo",
    prompt: `Merge check: ${id}.\n${newCheckPrompt(id)}\n`,
  };
}

/** The instructions the person edits. The check id stays on the saved prompt. */
export function mergeCheckPromptBody(prompt: string | undefined): string {
  if (!prompt) {
    return "";
  }
  return prompt.replace(MERGE_CHECK_HEADER, "");
}

/** Puts the check id back on the prompt after an edit. */
export function withMergeCheckHeader(current: string | undefined, body: string): string {
  const match = current?.match(MERGE_CHECK_HEADER);
  if (!match) {
    return body;
  }
  const header = match[0].endsWith("\n") ? match[0] : `${match[0]}\n`;
  return `${header}${body}`;
}

function mergeCheckId(prompt: string | undefined): string | null {
  return prompt?.match(MERGE_CHECK_HEADER)?.[1] ?? null;
}

function isCombinedReviewPrompt(prompt: string | undefined): boolean {
  return Boolean(prompt && prompt.includes("report_merge_check") && parseEnabledChecks(prompt));
}

function checkStepsFromReview(review: PlanningReviewStep): PlanningReviewStep[] {
  const prompt = review.prompt ?? "";
  const enabled = new Set(parseEnabledChecks(prompt) ?? []);
  const sections = sectionsIn(prompt);
  const shared = sharedContext(prompt, sections);
  return sections
    .filter((section) => enabled.has(section.id))
    .map((section) => ({
      name: section.label,
      type: "prompt" as const,
      workingDirectory: review.workingDirectory ?? "repo",
      prompt: checkPrompt(section.id, shared, section.body),
    }));
}

function checkPrompt(id: string, shared: string, body: string): string {
  return [
    `Merge check: ${id}.`,
    shared,
    "",
    "Report this check with the report_merge_check tool.",
    "Call the tool once.",
    `check is ${id}.`,
    "score is an integer from 1 to 3.",
    "summary is one sentence.",
    "Do not write a file.",
    "",
    body,
    "",
  ].join("\n");
}

function sharedContext(prompt: string, sections: readonly { marker: string }[]): string {
  const first = sections[0];
  if (!first) {
    return "";
  }
  const head = prompt.slice(0, prompt.indexOf(first.marker));
  return head
    .replace(/^Report each enabled check[\s\S]*?Do not write a file\.\n*/m, "")
    .replace(/^Enabled checks:.*\n?/m, "")
    .trim();
}

function sectionsIn(prompt: string): Array<{ id: MergeConfidenceCheck; label: string; marker: string; body: string }> {
  const found = CHECK_SECTIONS.flatMap((section) => {
    const start = prompt.indexOf(section.marker);
    return start < 0 ? [] : [{ ...section, start }];
  }).sort((left, right) => left.start - right.start);

  return found.map((section, index) => {
    const next = found[index + 1];
    const end = next ? next.start : prompt.length;
    const raw = prompt.slice(section.start + section.marker.length, end).trim();
    return { id: section.id, label: section.label, marker: section.marker, body: raw };
  });
}

function nextCheckId(steps: readonly PlanningReviewStep[]): string {
  const used = new Set(steps.map((step) => mergeCheckId(step.prompt)).filter((id): id is string => Boolean(id)));
  if (!used.has("new-check")) {
    return "new-check";
  }
  let suffix = 2;
  while (used.has(`new-check-${suffix}`)) {
    suffix += 1;
  }
  return `new-check-${suffix}`;
}
