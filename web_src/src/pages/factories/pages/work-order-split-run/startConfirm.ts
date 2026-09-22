import { draftReadiness, type DraftReadinessInput, type DraftReadinessTone } from "../../lib/draftReadiness";

export const START_CONFIRM_STORAGE_KEY = "sp:refine:skip-start-confirm";

export const START_CONFIRM_COPY = {
  title: "Start implementation?",
  skip: "Do not ask again",
  cancel: "Keep refining",
  confirm: "Start anyway",
  missing:
    "The agent did not finish the analysis and did not create a plan. Wait for the analysis to finish for better results.",
  low: "This task is not clear enough for an agent to plan it. Add more context in the chat before you start for better results.",
  mid: "This task is clear enough to start, but an agent may need steering. Refine the task in the chat, or start if you accept the risk.",
  working: "The agent is still working on this task. Wait until the analysis finishes. Then start for better results.",
} as const;

export type StartConfirmTone = "missing" | "low" | "mid" | "working";

export type StartConfirmScores = Pick<DraftReadinessInput, "clarity" | "confidence">;

export type StartConfirmInput = StartConfirmScores & {
  agentWorking?: boolean;
};

const CONFIRM_TONE: Record<DraftReadinessTone, StartConfirmTone | undefined> = {
  analyzing: "missing",
  pending: "missing",
  blocked: "low",
  caution: "mid",
  ready: undefined,
};

export function startConfirmTone(input: StartConfirmInput): StartConfirmTone | undefined {
  if (input.agentWorking) {
    return "working";
  }
  return CONFIRM_TONE[draftReadiness(input).tone];
}

export function startConfirmBody(input: StartConfirmInput): string | undefined {
  const tone = startConfirmTone(input);
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

export function needsStartConfirm(input: StartConfirmInput): boolean {
  if (startConfirmTone(input) === "working") {
    return true;
  }
  if (readSkipStartConfirm()) {
    return false;
  }
  return startConfirmTone(input) != null;
}

export function refineAgentIsWorking(machineStatus: string | undefined): boolean {
  return machineStatus === "starting" || machineStatus === "running";
}
