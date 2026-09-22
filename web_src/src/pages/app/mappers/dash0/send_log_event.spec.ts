import { describe, expect, it } from "bun:test";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { sendLogEventMapper } from "./send_log_event";

const NODE: NodeInfo = {
  id: "n1",
  name: "Send Log Event",
  componentName: "dash0.send_log_event",
  isCollapsed: false,
};

function makeDetailsContext(data: Record<string, unknown>): ExecutionDetailsContext {
  const execution: ExecutionInfo = {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    outputs: {
      default: [
        {
          type: "dash0.logEvent",
          timestamp: new Date().toISOString(),
          data,
        },
      ],
    },
  };
  return { nodes: [], node: NODE, execution };
}

describe("sendLogEventMapper.getExecutionDetails", () => {
  it("keeps attributes after dataset fields", () => {
    const details = sendLogEventMapper.getExecutionDetails(
      makeDetailsContext({
        sent: true,
        severityText: "ERROR",
        body: "disk full",
        eventName: "host.disk",
        serviceName: "api",
        dataset: "prod",
        attributes: { host: "box-1" },
      }),
    );

    expect(Object.keys(details)).toEqual([
      "Sent At",
      "Status",
      "Severity",
      "Body",
      "Event Name",
      "Service Name",
      "Dataset",
      "Attributes",
    ]);
  });
});
