import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { deleteIncidentMapper } from "./delete_incident";
import { eventStateRegistry } from "./index";

function buildNode(): NodeInfo {
  return {
    id: "n1",
    name: "Del",
    componentName: "jira.deleteIncident",
    isCollapsed: false,
    configuration: { issue: "IT-9" },
    metadata: {},
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "e1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  };
}

describe("deleteIncidentMapper", () => {
  it("getExecutionDetails shows Deleted true from payload", () => {
    const ctx: ExecutionDetailsContext = {
      nodes: [],
      node: buildNode(),
      execution: buildExecution({
        outputs: {
          default: [
            {
              type: "jira.incident.deleted",
              timestamp: new Date().toISOString(),
              data: { deleted: true },
            },
          ],
        },
      }),
    };
    expect(deleteIncidentMapper.getExecutionDetails(ctx)["Deleted"]).toBe("true");
    expect(deleteIncidentMapper.getExecutionDetails(ctx)["Issue id"]).toBeUndefined();
  });

  it("event state registry maps success to deleted", () => {
    const exec = buildExecution({ result: "RESULT_PASSED" });
    expect(eventStateRegistry.deleteIncident.getState(exec)).toBe("deleted");
  });
});
