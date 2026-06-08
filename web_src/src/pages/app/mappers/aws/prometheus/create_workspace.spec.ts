import { describe, expect, it } from "vitest";

import { createWorkspaceMapper } from "./create_workspace";
import { buildComponentCtx, buildDetailsCtx, buildOutput } from "./workspace_test_helpers";

describe("createWorkspaceMapper.props", () => {
  it("includes alias, region, and KMS metadata", () => {
    const props = createWorkspaceMapper.props(
      buildComponentCtx({
        componentName: "prometheus.createWorkspace",
        configuration: {
          alias: "metrics",
          region: "us-east-1",
          kmsKeyArn: "arn:aws:kms:us-east-1:123456789012:key/key-1",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "tag", label: "metrics" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
        expect.objectContaining({ icon: "key", label: "Custom KMS key" }),
      ]),
    );
  });
});

describe("createWorkspaceMapper.getExecutionDetails", () => {
  it("maps create workspace output", () => {
    const details = createWorkspaceMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [
              buildOutput({
                workspace: {
                  workspaceId: "ws-abc123",
                  arn: "arn:aws:aps:us-east-1:123456789012:workspace/ws-abc123",
                  status: { statusCode: "CREATING" },
                },
              }),
            ],
          },
        },
      }),
    );

    expect(details["Workspace ID"]).toBe("ws-abc123");
    expect(details.Status).toBe("CREATING");
  });
});
