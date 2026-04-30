import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { defaultStateFunction } from "../stateRegistry";
import type { EventStateRegistry, ExecutionInfo, OutputPayload } from "../types";

function silenceStateFromExecution(execution: ExecutionInfo): string | undefined {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const data = outputs?.default?.[0]?.data as { status?: { state?: string } } | undefined;
  const raw = data?.status?.state;
  if (typeof raw !== "string") {
    return undefined;
  }
  const normalized = raw.trim().toLowerCase();
  return normalized || undefined;
}

/**
 * Badge state for Get Silence: show Alertmanager silence status (active, pending, expired)
 * instead of a generic "fetched" label.
 */
export const getSilenceEventStateRegistry: EventStateRegistry = {
  stateMap: {
    ...DEFAULT_EVENT_STATE_MAP,
    active: { ...DEFAULT_EVENT_STATE_MAP.success },
    pending: { ...DEFAULT_EVENT_STATE_MAP.queued },
    expired: { ...DEFAULT_EVENT_STATE_MAP.neutral },
  },
  getState: (execution) => {
    const base = defaultStateFunction(execution);
    if (base !== "success") {
      return base;
    }
    return silenceStateFromExecution(execution) ?? "active";
  },
};
