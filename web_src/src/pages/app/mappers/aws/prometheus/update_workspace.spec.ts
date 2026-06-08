import { describe, expect, it } from "vitest";

import { updateWorkspaceMapper } from "./update_workspace";
import { buildDetailsCtx, buildOutput } from "./workspace_test_helpers";

describe("updateWorkspaceMapper.getExecutionDetails", () => {
  it("maps update workspace output", () => {
    const details = updateWorkspaceMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [buildOutput({ workspaceId: "ws-abc123", alias: "metrics", updated: true })],
          },
        },
      }),
    );

    expect(details).toEqual({
      "Workspace ID": "ws-abc123",
      Alias: "metrics",
      Status: "Updated",
    });
  });
});
