export const START_CONFIRM_STORAGE_KEY = "sp:refine:skip-start-confirm";
const READY_SCORE = 5;

export const START_CONFIRM_COPY = {
  title: "Start implementation?",
  skip: "Do not ask again",
  cancel: "Keep refining",
  confirm: "Start anyway",
  missing:
    "The agent did not finish the analysis and did not create a plan. Wait for the analysis to finish for better results.",
  low: "This request is not clear enough, and the agent could not create a plan. Consider adding more context in the chat before you start for better results.",
  mid: "Even though this task is clear enough to start, a few more refinements in the chat will improve the result.",
} as const;

export type StartConfirmTone = "missing" | "low" | "mid";

export function startConfirmTone(score?: number): StartConfirmTone | undefined {
  if (score == null) {
    return "missing";
  }
  if (score < 3) {
    return "low";
  }
  if (score < READY_SCORE) {
    return "mid";
  }
  return undefined;
}

export function startConfirmBody(score?: number): string | undefined {
  const tone = startConfirmTone(score);
  return tone ? START_CONFIRM_COPY[tone] : undefined;
}

export function readSkipStartConfirm(): boolean {
  try {
    return window.localStorage.getItem(START_CONFIRM_STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

export function persistSkipStartConfirm(): void {
  try {
    window.localStorage.setItem(START_CONFIRM_STORAGE_KEY, "1");
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}

export function needsStartConfirm(score?: number): boolean {
  if (readSkipStartConfirm()) {
    return false;
  }
  return startConfirmTone(score) != null;
}
