import { describe, expect, it } from "bun:test";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getCheckRuleMapper } from "./get_check_rule";

const NODE: NodeInfo = {
  id: "n1",
  name: "Get Check Rule",
  componentName: "dash0.get_check_rule",
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
          type: "dash0.checkRule",
          timestamp: new Date().toISOString(),
          data,
        },
      ],
    },
  };
  return { nodes: [], node: NODE, execution };
}

describe("getCheckRuleMapper.getExecutionDetails", () => {
  it("keeps labels after interval and status fields", () => {
    const details = getCheckRuleMapper.getExecutionDetails(
      makeDetailsContext({
        name: "High error rate",
        id: "rule-1",
        expression: "error_rate > 0.1",
        thresholds: { degraded: 0.05, critical: 0.1 },
        interval: "1m",
        for: "5m",
        keepFiringFor: "10m",
        enabled: true,
        labels: { team: "sre" },
      }),
    );

    expect(Object.keys(details)).toEqual([
      "Fetched At",
      "Name",
      "ID",
      "Expression",
      "Thresholds",
      "Interval",
      "For",
      "Keep Firing For",
      "Enabled",
      "Labels",
    ]);
  });
});
