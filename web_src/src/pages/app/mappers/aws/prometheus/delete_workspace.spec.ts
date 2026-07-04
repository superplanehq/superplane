import { describe, expect, it } from "vitest";

import { buildComponentCtx, buildDetailsCtx, buildOutput } from "./common";
import { deleteWorkspaceMapper } from "./delete_workspace";

describe("deleteWorkspaceMapper.props", () => {
  it("shows the workspace alias in metadata", () => {
    const props = deleteWorkspaceMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", workspace: "ws-abc123" },
        metadata: { workspaceAlias: "metrics" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "activity", label: "metrics" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
      ]),
    );
    expect(props.metadata).not.toEqual(expect.arrayContaining([expect.objectContaining({ label: "ws-abc123" })]));
  });
});

describe("deleteWorkspaceMapper.getExecutionDetails", () => {
  it("maps delete workspace output", () => {
    const details = deleteWorkspaceMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [buildOutput({ workspaceId: "ws-abc123", alias: "metrics", deleted: true })],
          },
        },
      }),
    );

    expect(details).toEqual({
      "Deleted At": new Date("2026-06-08T09:01:00Z").toLocaleString(),
      Alias: "metrics",
      Status: "Deleted",
    });
    expect(details["Workspace ID"]).toBeUndefined();
  });
});
