import { describe, expect, it } from "vitest";

import { buildComponentCtx, buildDetailsCtx, buildOutput } from "./common";
import { deleteRuleGroupNamespaceMapper } from "./delete_rule_group_namespace";

describe("deleteRuleGroupNamespaceMapper.props", () => {
  it("shows the namespace, workspace alias, and region in metadata", () => {
    const props = deleteRuleGroupNamespaceMapper.props(
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

describe("deleteRuleGroupNamespaceMapper.getExecutionDetails", () => {
  it("maps delete rule group namespace output", () => {
    const details = deleteRuleGroupNamespaceMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [
              buildOutput({
                workspaceId: "ws-abc123",
                namespace: "example-alerts",
                deleted: true,
              }),
            ],
          },
        },
      }),
    );

    expect(Object.keys(details)[0]).toBe("Deleted At");
    expect(details["Deleted At"]).toBe(new Date("2026-06-08T09:01:00Z").toLocaleString());
    expect(details.Name).toBe("example-alerts");
    expect(details.Status).toBe("Deleted");
  });

  it("falls back to configuration when there is no output yet", () => {
    const details = deleteRuleGroupNamespaceMapper.getExecutionDetails(
      buildDetailsCtx({
        node: { configuration: { namespace: "example-alerts", workspace: "ws-abc123" } },
        execution: { outputs: undefined },
      }),
    );

    expect(details.Name).toBe("example-alerts");
    expect(details.Status).toBe("-");
  });
});
