import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { mergePullRequestMapper } from "./merge_pull_request";

describe("mergePullRequestMapper", () => {
  it("shows a curated merge summary", () => {
    const details = mergePullRequestMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          repository: "hello",
          pullNumber: "42",
          mergeMethod: "squash",
        },
        outputs: {
          default: [
            {
              data: {
                sha: "0e98bc41ab56cee9ff17883607b56f96e7814c98",
                merged: true,
                message: "Pull Request successfully merged",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      Repository: "hello",
      "Pull Request": "#42",
      "Pull Request URL": "https://github.com/testhq/hello/pull/42",
      "Merge method": "Squash",
      Merged: "Yes",
      SHA: "0e98bc4",
      Message: "Pull Request successfully merged",
    });
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Merge pull request",
    componentName: "github.mergePullRequest",
    isCollapsed: false,
    metadata: {
      repository: {
        id: "123456",
        name: "hello",
        url: "https://github.com/testhq/hello",
      },
    },
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
