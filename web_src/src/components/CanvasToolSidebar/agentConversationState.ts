import { useCallback, useMemo, useState } from "react";
import type {
  GradingEntry,
  IterationEntry,
  OutcomePhase,
  OutcomeState,
} from "@/components/AgentSidebar/widgets/OutcomeProgressWidget";
import type { RubricCategory } from "@/components/AgentSidebar/widgets/parser";
import type { AgentMessage } from "./types";

export type OutcomeEvaluationPayload = {
  iteration: number;
  result?: string;
  explanation?: string;
};

export function useConversationMessages(
  data: { pages: Array<{ messages: AgentMessage[] }> } | undefined,
): AgentMessage[] {
  return useMemo(
    () =>
      data?.pages
        .slice()
        .reverse()
        .flatMap((page) => page.messages) ?? [],
    [data],
  );
}

export function useThinkingIndicator(messages: AgentMessage[], status: string): boolean {
  const hasRunningTool = useMemo(() => hasActiveTool(messages), [messages]);
  return status === "streaming" && !hasRunningTool;
}

function hasActiveTool(messages: AgentMessage[]): boolean {
  const activeToolIds = new Set<string>();

  for (const message of messages) {
    if (message.role !== "tool") continue;

    const key = message.toolCallId || message.id || message.toolName;
    if (!key) continue;

    if (message.toolStatus === "started") {
      activeToolIds.add(key);
      continue;
    }

    if (message.toolStatus === "finished") {
      activeToolIds.delete(key);
    }
  }

  return activeToolIds.size > 0;
}

export function useStoredOutcomeState(
  chatId: string,
): [OutcomeState | null, (update: OutcomeState | null | ((prev: OutcomeState | null) => OutcomeState | null)) => void] {
  const [outcomeState, setOutcomeStateRaw] = useState<OutcomeState | null>(() => {
    try {
      const stored = sessionStorage.getItem(`outcome-${chatId}`);
      return stored ? JSON.parse(stored) : null;
    } catch {
      return null;
    }
  });

  const setOutcomeState = useCallback(
    (update: OutcomeState | null | ((prev: OutcomeState | null) => OutcomeState | null)) => {
      setOutcomeStateRaw((prev) => {
        const next = typeof update === "function" ? update(prev) : update;
        if (next) {
          sessionStorage.setItem(`outcome-${chatId}`, JSON.stringify(next));
        } else {
          sessionStorage.removeItem(`outcome-${chatId}`);
        }
        return next;
      });
    },
    [chatId],
  );

  return [outcomeState, setOutcomeState];
}

export function createWebsocketCallbacks(
  setStatus: (value: string) => void,
  setError: (value: string | null) => void,
  setOutcomeState: (update: OutcomeState | null | ((prev: OutcomeState | null) => OutcomeState | null)) => void,
  setNotice?: (value: string | null) => void,
) {
  return {
    onPersistedMessage: (message: AgentMessage) => {
      if (message.content?.includes("published") || message.content?.includes("discarded")) {
        setOutcomeState(null);
      }
    },
    onStatusChange: (next: string, error?: string) => {
      setStatus(next || "idle");
      setError(error ?? null);
      setNotice?.(null); // turn boundary clears any stale notice
    },
    onNotice: (message: string) => {
      setNotice?.(message || "The agent hit a recoverable error and is retrying.");
    },
    onOutcomeEvent: (phase: "start" | "end", evaluation: OutcomeEvaluationPayload) => {
      setOutcomeState((prev) => {
        if (!prev) return prev;
        return phase === "start" ? applyOutcomeStart(prev) : applyOutcomeEnd(prev, evaluation);
      });
    },
  };
}

export function buildRubricText(rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }): string {
  if (rubric.categories && rubric.categories.length > 0) {
    const sections = rubric.categories
      .map((category) => {
        const body = category.body || category.criteria.map((criterion) => `- ${criterion.text}`).join("\n");
        return `## ${category.heading}\n${body}`;
      })
      .join("\n\n");
    return `# ${rubric.title}\n\n${sections}`;
  }

  return `# ${rubric.title}\n\n${rubric.criteria.map((criterion) => `- ${criterion}`).join("\n")}`;
}

export function createInitialOutcomeState(rubric: {
  title: string;
  criteria: string[];
  categories?: RubricCategory[];
}): OutcomeState {
  return {
    title: rubric.title,
    criteria: rubric.criteria.map((criterion) => ({ text: criterion })),
    categories: rubric.categories,
    iteration: 1,
    maxIterations: 3,
    phase: "building",
    log: [{ phase: "building" }],
  };
}

export function isOutcomeActive(outcomeState: OutcomeState | null): boolean {
  if (!outcomeState) {
    return false;
  }

  return outcomeState.phase !== "passed" && outcomeState.phase !== "exhausted" && outcomeState.phase !== "failed";
}

export function statusLabel(status: string): string {
  switch (status) {
    case "streaming":
      return "Agent is running...";
    case "failed":
      return "Message failed. Try again.";
    case "terminated":
      return "Session ended";
    default:
      return "Ready";
  }
}

function applyOutcomeStart(prev: OutcomeState): OutcomeState {
  const updatedLog = [...prev.log];
  const lastEntry = updatedLog[updatedLog.length - 1];

  if (lastEntry && "phase" in lastEntry && (lastEntry as IterationEntry).phase === "building") {
    updatedLog[updatedLog.length - 1] = { phase: "finished" };
  }

  updatedLog.push({ phase: "grading" });
  return {
    ...prev,
    phase: "grading" as OutcomePhase,
    log: updatedLog,
  };
}

function updateGradingLog(log: OutcomeState["log"], evaluation: OutcomeEvaluationPayload): OutcomeState["log"] {
  const updatedLog = [...log];

  for (let index = updatedLog.length - 1; index >= 0; index--) {
    const entry = updatedLog[index] as GradingEntry;
    if (entry.phase !== "grading") {
      continue;
    }

    updatedLog[index] = {
      phase: evaluation.result === "satisfied" ? "satisfied" : "needs_revision",
      explanation: evaluation.explanation,
    };
    break;
  }

  return updatedLog;
}

function applyOutcomeEnd(prev: OutcomeState, evaluation: OutcomeEvaluationPayload): OutcomeState {
  if (!evaluation.result) {
    return prev;
  }

  const updatedLog = updateGradingLog(prev.log, evaluation);

  switch (evaluation.result) {
    case "satisfied":
      return { ...prev, phase: "passed" as OutcomePhase, log: updatedLog };
    case "max_iterations_reached":
      return { ...prev, phase: "exhausted" as OutcomePhase, log: updatedLog };
    case "failed":
    case "interrupted":
      return { ...prev, phase: "failed" as OutcomePhase, log: updatedLog };
  }

  const nextIteration = prev.iteration + 1;
  if (nextIteration > prev.maxIterations) {
    return { ...prev, phase: "exhausted" as OutcomePhase, log: updatedLog };
  }

  updatedLog.push({ phase: "building" });
  return {
    ...prev,
    iteration: nextIteration,
    phase: "building" as OutcomePhase,
    log: updatedLog,
  };
}
