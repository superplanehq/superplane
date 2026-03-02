import { EventStateRegistry, OutputPayload, StateFunction } from "../types";
import { DEFAULT_EVENT_STATE_MAP, EventStateMap } from "@/ui/componentBase";
import { defaultStateFunction } from "../stateRegistry";

interface ExecuteCommandOutput {
  exitCode?: number;
}

export const EXECUTE_COMMAND_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  executed: DEFAULT_EVENT_STATE_MAP.success,
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
};

export const executeCommandStateFunction: StateFunction = (execution) => {
  if (!execution) return "neutral";

  const defaultState = defaultStateFunction(execution);
  if (defaultState !== "success") {
    return defaultState;
  }

  const outputs = execution.outputs as
    | { success?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] }
    | undefined;

  if (outputs?.failed?.length) {
    return "failed";
  }

  const payload =
    (outputs?.success?.[0]?.data as ExecuteCommandOutput | undefined) ??
    (outputs?.failed?.[0]?.data as ExecuteCommandOutput | undefined) ??
    (outputs?.default?.[0]?.data as ExecuteCommandOutput | undefined);

  if (typeof payload?.exitCode === "number" && payload.exitCode !== 0) {
    return "failed";
  }

  return "executed";
};

export const EXECUTE_COMMAND_STATE_REGISTRY: EventStateRegistry = {
  stateMap: EXECUTE_COMMAND_STATE_MAP,
  getState: executeCommandStateFunction,
};
