import { WorkflowsWorkflowNodeExecution } from "@/api-client";
import { DEFAULT_EVENT_STATE_MAP, EventState } from "@/ui/componentBase";
import { EventStateRegistry, StateFunction } from "./types";

/**
 * Default state logic function used by most components
 */
export const defaultStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return "running";
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return "success";
  }

  return "failed";
};

/**
 * Default state registry used by most components
 */
export const DEFAULT_STATE_REGISTRY: EventStateRegistry = {
  stateMap: DEFAULT_EVENT_STATE_MAP,
  getState: defaultStateFunction,
};
