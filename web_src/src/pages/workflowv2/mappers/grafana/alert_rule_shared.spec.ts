import { describe, expect, it } from "vitest";

import { buildAlertRuleMetadata } from "./alert_rule_shared";
import type { NodeInfo } from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Alert Rule",
    componentName: "grafana.updateAlertRule",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

describe("buildAlertRuleMetadata", () => {
  it("ignores empty configuration titles and falls back to alertRule", () => {
    const metadata = buildAlertRuleMetadata(
      buildNode({
        configuration: {
          title: "",
          alertRule: "rule-123",
        },
      }),
      { includeUid: true },
    );

    expect(metadata).toEqual([expect.objectContaining({ icon: "hash", label: "rule-123" })]);
  });

  it("trims whitespace titles before rendering metadata", () => {
    const metadata = buildAlertRuleMetadata(
      buildNode({
        configuration: {
          title: "  Production Alert  ",
        },
      }),
    );

    expect(metadata).toEqual([expect.objectContaining({ icon: "bell", label: "Production Alert" })]);
  });
});
