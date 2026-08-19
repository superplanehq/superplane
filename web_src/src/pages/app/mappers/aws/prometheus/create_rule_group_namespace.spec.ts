import { describe, expect, it } from "vitest";

import { buildComponentCtx, buildDetailsCtx, buildOutput } from "./common";
import { createRuleGroupNamespaceMapper } from "./create_rule_group_namespace";

describe("createRuleGroupNamespaceMapper.props", () => {
  it("includes name, workspace, and region metadata", () => {
    const props = createRuleGroupNamespaceMapper.props(
      buildComponentCtx({
        componentName: "prometheus.createRuleGroupNamespace",
        configuration: {
          name: "example-alerts",
          workspace: "ws-abc123",
          region: "us-east-1",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "tag", label: "example-alerts" }),
        expect.objectContaining({ icon: "activity", label: "ws-abc123" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
      ]),
    );
  });

  it("shows the workspace alias when available", () => {
    const props = createRuleGroupNamespaceMapper.props(
      buildComponentCtx({
        configuration: { name: "example-alerts", workspace: "ws-abc123", region: "us-east-1" },
        metadata: { workspaceAlias: "metrics" },
      }),
    );

    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "metrics" })]));
  });
});

describe("createRuleGroupNamespaceMapper.getExecutionDetails", () => {
  it("maps create rule group namespace output", () => {
    const details = createRuleGroupNamespaceMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [
              buildOutput({
                workspaceId: "ws-abc123",
                namespace: {
                  name: "example-alerts",
                  arn: "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/example-alerts",
                  status: { statusCode: "CREATING" },
                },
              }),
            ],
          },
        },
      }),
    );

    expect(Object.keys(details)[0]).toBe("Created At");
    expect(details["Created At"]).toBe(new Date("2026-06-08T09:00:00Z").toLocaleString());
    expect(details.Name).toBe("example-alerts");
    expect(details.Status).toBe("CREATING");
  });
});
