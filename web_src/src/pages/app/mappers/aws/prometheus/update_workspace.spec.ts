import { describe, expect, it } from "vitest";

import { buildComponentCtx, buildDetailsCtx, buildOutput } from "./common";
import { updateWorkspaceMapper } from "./update_workspace";

describe("updateWorkspaceMapper.props", () => {
  it("shows the current and new workspace aliases in metadata", () => {
    const props = updateWorkspaceMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", workspace: "ws-abc123", alias: "metrics-v2" },
        metadata: { workspaceAlias: "metrics" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "activity", label: "Current: metrics" }),
        expect.objectContaining({ icon: "tag", label: "New: metrics-v2" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
      ]),
    );
    expect(props.metadata).not.toEqual(expect.arrayContaining([expect.objectContaining({ label: "ws-abc123" })]));
  });
});

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
      "Updated At": new Date("2026-06-08T09:01:00Z").toLocaleString(),
      Alias: "metrics",
      Status: "Updated",
    });
    expect(details["Workspace ID"]).toBeUndefined();
  });
});
