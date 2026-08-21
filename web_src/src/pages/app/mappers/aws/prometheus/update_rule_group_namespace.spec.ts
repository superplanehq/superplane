import { describe, expect, it } from "vitest";

import { buildComponentCtx, buildDetailsCtx, buildOutput } from "./common";
import { updateRuleGroupNamespaceMapper } from "./update_rule_group_namespace";

describe("updateRuleGroupNamespaceMapper.props", () => {
  it("shows the namespace, workspace alias, and region in metadata", () => {
    const props = updateRuleGroupNamespaceMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", workspace: "ws-abc123", namespace: "example-alerts" },
        metadata: { workspaceAlias: "metrics" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "tag", label: "example-alerts" }),
        expect.objectContaining({ icon: "activity", label: "metrics" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
      ]),
    );
  });
});

describe("updateRuleGroupNamespaceMapper.getExecutionDetails", () => {
  it("maps update rule group namespace output", () => {
    const details = updateRuleGroupNamespaceMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [
              buildOutput({
                workspaceId: "ws-abc123",
                namespace: {
                  name: "example-alerts",
                  arn: "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/example-alerts",
                  status: { statusCode: "UPDATING" },
                },
              }),
            ],
          },
        },
      }),
    );

    expect(Object.keys(details)[0]).toBe("Updated At");
    expect(details["Updated At"]).toBe(new Date("2026-06-08T09:01:00Z").toLocaleString());
    expect(details.Name).toBe("example-alerts");
    expect(details.Status).toBe("UPDATING");
  });
});
