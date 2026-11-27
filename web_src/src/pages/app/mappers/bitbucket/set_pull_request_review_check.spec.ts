import { describe, expect, it } from "bun:test";
import type { ExecutionInfo, NodeInfo } from "../types";
import { componentMappers, eventStateRegistry } from "./index";

function execution(status: string): ExecutionInfo {
  return {
    id: "check",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: { repository: "acme/widgets", pullRequestId: "42" },
    rootEvent: undefined,
    outputs: {
      default: [
        {
          data: {
            status,
            state: "INPROGRESS",
            reason: "approval_commit_unverified",
            reviewers: [{ uuid: "reviewer-id", reason: "approval_commit_unverified" }],
          },
        },
      ],
    },
  };
}

describe("Bitbucket review check mapper", () => {
  it.each(["registered", "superseded", "skipped"])("keeps %s separate from an applied decision", (status) => {
    expect(eventStateRegistry.setPullRequestReviewCheck.getState(execution(status))).toBe(status);
  });

  it("shows why a provider approval remains unresolved", () => {
    const details = componentMappers.setPullRequestReviewCheck.getExecutionDetails!({
      nodes: [],
      node: { id: "check", configuration: {} } as NodeInfo,
      execution: execution("applied"),
    });
    expect(details.Reason).toBe("approval commit unverified");
    expect(details.Reviewers).toContain("reviewer-id");
    expect(details["Commit status"]).toBe("INPROGRESS");
  });
});
