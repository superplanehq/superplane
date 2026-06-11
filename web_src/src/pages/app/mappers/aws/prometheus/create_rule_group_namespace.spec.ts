import { describe, expect, it } from "vitest";

import { eventStateRegistry } from "..";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./common";
import { createRuleGroupNamespaceMapper } from "./create_rule_group_namespace";

describe("createRuleGroupNamespaceMapper.props", () => {
  it("includes namespace, workspace alias, and region metadata", () => {
    const props = createRuleGroupNamespaceMapper.props(
      buildComponentCtx({
        componentName: "prometheus.createRuleGroupNamespace",
        configuration: {
          region: "us-east-1",
          workspace: "ws-abc123",
          name: "application-rules",
        },
        metadata: { workspaceAlias: "metrics" },
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

describe("createRuleGroupNamespaceMapper.getExecutionDetails", () => {
  it("maps create namespace output", () => {
    const details = createRuleGroupNamespaceMapper.getExecutionDetails(
      buildDetailsCtx({
        node: {
          configuration: { name: "application-rules" },
        },
        execution: {
          outputs: {
            default: [
              buildOutput({
                ruleGroupNamespace: {
                  name: "application-rules",
                  arn: "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/application-rules",
                  status: { statusCode: "CREATING" },
                },
              }),
            ],
          },
        },
      }),
    );

    expect(Object.keys(details)[0]).toBe("Created At");
    expect(details).toEqual({
      "Created At": new Date("2026-06-08T09:00:00Z").toLocaleString(),
      Namespace: "application-rules",
    });
  });
});

describe("eventStateRegistry.prometheus.createRuleGroupNamespace", () => {
  it("maps success to created", () => {
    expect(eventStateRegistry["prometheus.createRuleGroupNamespace"].getState(buildExecution())).toBe("created");
  });
});
