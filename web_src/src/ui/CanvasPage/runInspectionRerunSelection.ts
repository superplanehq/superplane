import type { SidebarEvent } from "../componentSidebar/types";

const RERUN_SELECTION_ATTEMPTS = 5;
const RERUN_SELECTION_RETRY_DELAY_MS = 500;

type CreatedRerunSelectionOptions = {
  eventId: string;
  triggerNodeId?: string;
  selectedNodeId?: string | null;
  fetchRunId?: (event: SidebarEvent, options?: { maxPages?: number }) => Promise<string | null>;
  selectRun?: (runId: string, options?: { nodeId?: string }) => void;
  attempts?: number;
  retryDelayMs?: number;
};

export async function selectCreatedRerun({
  eventId,
  triggerNodeId,
  selectedNodeId,
  fetchRunId,
  selectRun,
  attempts = RERUN_SELECTION_ATTEMPTS,
  retryDelayMs = RERUN_SELECTION_RETRY_DELAY_MS,
}: CreatedRerunSelectionOptions) {
  if (!eventId || !triggerNodeId || !fetchRunId || !selectRun) return;

  const lookupEvent = buildRerunLookupEvent(eventId, triggerNodeId);
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const runId = await fetchRunId(lookupEvent, { maxPages: 1 });
    if (runId) {
      selectRun(runId, { nodeId: selectedNodeId ?? triggerNodeId });
      return;
    }

    if (attempt < attempts - 1) {
      await wait(retryDelayMs);
    }
  }
}

function buildRerunLookupEvent(eventId: string, triggerNodeId: string): SidebarEvent {
  return {
    id: eventId,
    title: "Re-emitted event",
    state: "running",
    isOpen: false,
    nodeId: triggerNodeId,
    triggerEventId: eventId,
    kind: "trigger",
  };
}

function wait(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
