import { describe, expect, it } from "vitest";

import { eventStateRegistry } from "..";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./common";
import { updateRuleGroupNamespaceMapper } from "./update_rule_group_namespace";

describe("updateRuleGroupNamespaceMapper.props", () => {
  it("includes namespace, workspace alias, and region metadata", () => {
    const props = updateRuleGroupNamespaceMapper.props(
      buildComponentCtx({
        componentName: "prometheus.updateRuleGroupNamespace",
        configuration: {
          region: "us-east-1",
          workspace: "ws-abc123",
          namespace: "application-rules",
        },
        metadata: { workspaceAlias: "metrics", namespace: "application-rules" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "file-text", label: "application-rules" }),
        expect.objectContaining({ icon: "activity", label: "metrics" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
      ]),
    );
  });
});

describe("updateRuleGroupNamespaceMapper.getExecutionDetails", () => {
  it("maps update namespace output", () => {
    const details = updateRuleGroupNamespaceMapper.getExecutionDetails(
      buildDetailsCtx({
        node: {
          metadata: { namespace: "application-rules" },
        },
        execution: {
          outputs: {
            default: [
              buildOutput({
                ruleGroupNamespace: {
                  name: "application-rules",
                  arn: "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/application-rules",
                  status: { statusCode: "UPDATING" },
                },
              }),
            ],
          },
        },
      }),
    );

    expect(details).toEqual({
      "Updated At": new Date("2026-06-08T09:01:00Z").toLocaleString(),
      Namespace: "application-rules",
    });
  });
});

describe("eventStateRegistry.prometheus.updateRuleGroupNamespace", () => {
  it("maps success to updated", () => {
    expect(eventStateRegistry["prometheus.updateRuleGroupNamespace"].getState(buildExecution())).toBe("updated");
  });
});
