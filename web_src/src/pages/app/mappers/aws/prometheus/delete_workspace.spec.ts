import { describe, expect, it } from "vitest";

import { deleteWorkspaceMapper } from "./delete_workspace";
import { buildDetailsCtx, buildOutput } from "./workspace_test_helpers";

describe("deleteWorkspaceMapper.getExecutionDetails", () => {
  it("maps delete workspace output", () => {
    const details = deleteWorkspaceMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [buildOutput({ workspaceId: "ws-abc123", deleted: true })],
          },
        },
      }),
    );

    expect(details).toEqual({
      "Workspace ID": "ws-abc123",
      Status: "Deleted",
    });
  });
});
