import { describe, expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";
import { runTaskMapper } from "./run_task";

describe("runTaskMapper.getExecutionDetails", () => {
  it("keeps the output fields in order", () => {
    const details = runTaskMapper.getExecutionDetails(
      buildDetailsContext({
        createdAt: "2026-05-21T12:00:00Z",
        updatedAt: "2026-05-21T12:01:00Z",
        outputs: {
          default: [
            {
              data: {
                tasks: [
                  {
                    taskArn: "arn:aws:ecs:us-east-1:123456789012:task/app/abc123",
                    clusterArn: "arn:aws:ecs:us-east-1:123456789012:cluster/app",
                    lastStatus: "RUNNING",
                  },
                ],
                failures: [{ reason: "RESOURCE:MEMORY" }],
              },
            },
          ],
        },
      }),
    );

    expect(Object.keys(details)).toEqual([
      "Started At",
      "Tasks Started",
      "Failures",
      "Task ARN",
      "Last Status",
      "ECS Console",
    ]);
    expect(details["Tasks Started"]).toBe("1");
    expect(details["Failures"]).toBe("1");
    expect(details["Task ARN"]).toBe("arn:aws:ecs:us-east-1:123456789012:task/app/abc123");
    expect(details["Last Status"]).toBe("RUNNING");
    expect(details["ECS Console"]).toBe("https://ecs.console.aws.amazon.com/ecs/v2/clusters/app/tasks/abc123");
  });

  it("keeps only the start time when the output is empty", () => {
    const details = runTaskMapper.getExecutionDetails(
      buildDetailsContext({
        createdAt: "2026-05-21T12:00:00Z",
        updatedAt: "",
      }),
    );

    expect(Object.keys(details)).toEqual(["Started At"]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Run task",
    componentName: "aws.ecs.runTask",
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
