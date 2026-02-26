const AI_BUILDER_STORAGE_KEY_PREFIX = "sp:canvas-ai-builder";

export type PersistedAiBuilderMessage = {
  id: string;
  role: "user" | "assistant";
  content: string;
};

export type PersistedAiBuilderProposal<TOperation> = {
  id: string;
  summary: string;
  operations: TOperation[];
};

export type PersistedAiBuilderState<TOperation> = {
  activeTab: "components" | "ai";
  messages: PersistedAiBuilderMessage[];
  pendingProposal: PersistedAiBuilderProposal<TOperation> | null;
};

function getAiBuilderStorageKey(canvasId?: string): string | null {
  if (!canvasId) {
    return null;
  }

  return `${AI_BUILDER_STORAGE_KEY_PREFIX}:${canvasId}`;
}

export function loadAiBuilderState<TOperation>(canvasId?: string): PersistedAiBuilderState<TOperation> | null {
  if (typeof window === "undefined") {
    return null;
  }

  const storageKey = getAiBuilderStorageKey(canvasId);
  if (!storageKey) {
    return null;
  }

  const rawState = window.localStorage.getItem(storageKey);
  if (!rawState) {
    return null;
  }

  try {
    const parsed = JSON.parse(rawState) as Partial<PersistedAiBuilderState<TOperation>>;
    const activeTab = parsed.activeTab === "ai" ? "ai" : "components";
    const messages = Array.isArray(parsed.messages)
      ? parsed.messages.filter(
          (message): message is PersistedAiBuilderMessage =>
            !!message &&
            typeof message.id === "string" &&
            (message.role === "user" || message.role === "assistant") &&
            typeof message.content === "string",
        )
      : [];
    const pendingProposal =
      parsed.pendingProposal &&
      typeof parsed.pendingProposal.id === "string" &&
      typeof parsed.pendingProposal.summary === "string" &&
      Array.isArray(parsed.pendingProposal.operations)
        ? parsed.pendingProposal
        : null;

    return {
      activeTab,
      messages,
      pendingProposal,
    };
  } catch (error) {
    console.warn("Failed to parse AI builder state from local storage:", error);
    return null;
  }
}

export function saveAiBuilderState<TOperation>(
  canvasId: string | undefined,
  state: PersistedAiBuilderState<TOperation>,
): void {
  if (typeof window === "undefined") {
    return;
  }

  const storageKey = getAiBuilderStorageKey(canvasId);
  if (!storageKey) {
    return;
  }

  try {
    window.localStorage.setItem(storageKey, JSON.stringify(state));
  } catch (error) {
    console.warn("Failed to save AI builder state to local storage:", error);
  }
}
