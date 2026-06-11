import { describe, expect, it } from "vitest";

import { eventStateRegistry } from "..";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./common";
import { getRuleGroupNamespaceMapper } from "./get_rule_group_namespace";

describe("getRuleGroupNamespaceMapper.props", () => {
  it("includes namespace, workspace alias, and region metadata", () => {
    const props = getRuleGroupNamespaceMapper.props(
      buildComponentCtx({
        componentName: "prometheus.getRuleGroupNamespace",
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

describe("getRuleGroupNamespaceMapper.getExecutionDetails", () => {
  it("maps get namespace output without rule YAML data", () => {
    const details = getRuleGroupNamespaceMapper.getExecutionDetails(
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
                  data: "Z3JvdXBzOiBbXQ==",
                  status: { statusCode: "ACTIVE" },
                },
              }),
            ],
          },
        },
      }),
    );

    expect(details).toEqual({
      "Retrieved At": new Date("2026-06-08T09:01:00Z").toLocaleString(),
      Namespace: "application-rules",
      Status: "ACTIVE",
    });
    expect(details.Data).toBeUndefined();
  });
});

describe("eventStateRegistry.prometheus.getRuleGroupNamespace", () => {
  it("maps success to retrieved", () => {
    expect(eventStateRegistry["prometheus.getRuleGroupNamespace"].getState(buildExecution())).toBe("retrieved");
  });
});
