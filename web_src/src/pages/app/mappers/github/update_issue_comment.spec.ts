import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { updateIssueCommentMapper } from "./update_issue_comment";

describe("updateIssueCommentMapper", () => {
  it("shows updated comment details", () => {
    const details = updateIssueCommentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          repository: "superplane",
          commentId: "1234567890",
          body: "Updated summary",
        },
        outputs: {
          default: [
            {
              data: {
                id: 1234567890,
                html_url: "https://github.com/superplanehq/superplane/issues/42#issuecomment-1234567890",
                updated_at: "2026-06-11T14:30:00Z",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Updated At": expect.any(String),
      URL: "https://github.com/superplanehq/superplane/issues/42#issuecomment-1234567890",
    });
  });

  it("handles missing outputs", () => {
    const details = updateIssueCommentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { repository: "superplane" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Update Issue Comment",
    componentName: "github.updateIssueComment",
    isCollapsed: false,
    metadata: {},
    configuration: execution.configuration ?? {},
  };

  return {
    node,
    execution: {
      id: "exec-1",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      configuration: execution.configuration ?? {},
      outputs: execution.outputs ?? {},
      metadata: {},
      resultMessage: "",
      resultReason: "",
      createdAt: "2026-06-11T12:00:00Z",
      updatedAt: "2026-06-11T14:30:00Z",
      childExecutions: [],
    },
  };
}
