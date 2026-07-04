import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getCombinedCommitStatusMapper } from "./get_combined_commit_status";

describe("getCombinedCommitStatusMapper", () => {
  it("summarizes combined commit status output", () => {
    const details = getCombinedCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              data: {
                state: "failure",
                sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
                total_count: 4,
                commit_url: "https://api.github.com/repos/acme/snaketoy/commits/d6f3c8a",
                repository_url: "https://api.github.com/repos/acme/snaketoy",
                statuses: [
                  { state: "success", context: "ci/build" },
                  { state: "failure", context: "ci/lint" },
                  { state: "error", context: "security/scan" },
                  { state: "pending", context: "deploy/preview" },
                ],
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      State: "failure",
      "Total statuses": "4",
      Successful: "1",
      Failed: "1",
      Errored: "1",
      Pending: "1",
      "First non-success status": "ci/lint",
      SHA: "d6f3c8a",
      "Commit URL": "https://api.github.com/repos/acme/snaketoy/commits/d6f3c8a",
      "Repository URL": "https://api.github.com/repos/acme/snaketoy",
    });
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Get combined status",
    componentName: "github.getCombinedCommitStatus",
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
