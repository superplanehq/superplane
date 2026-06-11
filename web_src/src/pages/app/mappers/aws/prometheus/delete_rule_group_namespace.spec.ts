import { describe, expect, it } from "vitest";

import { eventStateRegistry } from "..";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./common";
import { deleteRuleGroupNamespaceMapper } from "./delete_rule_group_namespace";

describe("deleteRuleGroupNamespaceMapper.props", () => {
  it("includes namespace, workspace alias, and region metadata", () => {
    const props = deleteRuleGroupNamespaceMapper.props(
      buildComponentCtx({
        componentName: "prometheus.deleteRuleGroupNamespace",
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

describe("deleteRuleGroupNamespaceMapper.getExecutionDetails", () => {
  it("maps delete namespace output", () => {
    const details = deleteRuleGroupNamespaceMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [buildOutput({ namespace: "application-rules", deleted: true })],
          },
        },
      }),
    );

    expect(details).toEqual({
      "Deleted At": new Date("2026-06-08T09:01:00Z").toLocaleString(),
      Namespace: "application-rules",
      Status: "Deleted",
    });
  });
});

describe("eventStateRegistry.prometheus.deleteRuleGroupNamespace", () => {
  it("maps success to deleted", () => {
    expect(eventStateRegistry["prometheus.deleteRuleGroupNamespace"].getState(buildExecution())).toBe("deleted");
  });
});
