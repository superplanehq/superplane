import { describe, expect, it } from "vitest";

import { getWorkspaceMapper } from "./get_workspace";
import { buildDetailsCtx, buildOutput } from "./workspace_test_helpers";

describe("getWorkspaceMapper.getExecutionDetails", () => {
  it("maps get workspace output", () => {
    const details = getWorkspaceMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [
              buildOutput({
                workspace: {
                  workspaceId: "ws-abc123",
                  alias: "metrics",
                  arn: "arn:aws:aps:us-east-1:123456789012:workspace/ws-abc123",
                  prometheusEndpoint: "https://aps-workspaces.us-east-1.amazonaws.com/workspaces/ws-abc123/api/v1/",
                  status: { statusCode: "ACTIVE" },
                },
              }),
            ],
          },
        },
      }),
    );

    expect(details["Workspace ID"]).toBe("ws-abc123");
    expect(details.Alias).toBe("metrics");
    expect(details.Status).toBe("ACTIVE");
    expect(details["Prometheus Endpoint"]).toContain("/workspaces/ws-abc123/");
  });
});
