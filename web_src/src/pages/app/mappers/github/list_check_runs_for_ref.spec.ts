import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { listCheckRunsForRefMapper } from "./list_check_runs_for_ref";

describe("listCheckRunsForRefMapper", () => {
  it("summarizes check run results for execution details", () => {
    const details = listCheckRunsForRefMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              data: {
                total_count: 3,
                check_runs: [
                  { name: "DCO", status: "completed", conclusion: "success" },
                  { name: "Cloudflare Pages", status: "completed", conclusion: "timed_out" },
                  { name: "Sourcery", status: "in_progress" },
                ],
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Total check runs": "3",
      Completed: "2",
      Successful: "1",
      "Not successful": "1",
      Pending: "1",
      "First non-green check": "Cloudflare Pages",
    });
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "List checks",
    componentName: "github.listCheckRunsForRef",
    isCollapsed: false,
  };

  return {
    nodes: [node],
    node,
    execution: {
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
      ...execution,
    },
  };
}
