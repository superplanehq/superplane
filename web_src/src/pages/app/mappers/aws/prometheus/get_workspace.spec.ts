import { describe, expect, it } from "vitest";

import { buildComponentCtx, buildDetailsCtx, buildOutput } from "./common";
import { getWorkspaceMapper } from "./get_workspace";

describe("getWorkspaceMapper.props", () => {
  it("shows the workspace alias in metadata", () => {
    const props = getWorkspaceMapper.props(
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

    expect(Object.keys(details)[0]).toBe("Retrieved At");
    expect(details["Retrieved At"]).toBe(new Date("2026-06-08T09:01:00Z").toLocaleString());
    expect(details["Workspace ID"]).toBeUndefined();
    expect(details.Alias).toBe("metrics");
    expect(details.Status).toBe("ACTIVE");
    expect(details["Prometheus Endpoint"]).toContain("/workspaces/ws-abc123/");
  });
});
