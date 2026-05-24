import { describe, expect, it } from "vitest";

import { createSilenceMapper } from "./create_silence";
import { truncate } from "./incident_shared";
import type { ComponentBaseContext, NodeInfo } from "../types";

function buildNode(componentName: string, overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Grafana Mapper",
    componentName,
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildComponentContext(componentName: string, overrides?: { node?: Partial<NodeInfo> }): ComponentBaseContext {
  const node = buildNode(componentName, overrides?.node);

  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: componentName,
      label: componentName,
      description: "",
      icon: "bolt",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
  };
}

describe("createSilenceMapper", () => {
  it("does not throw when configuration.comment is a non-string value", () => {
    const ctx = buildComponentContext("grafana.createSilence", {
      node: { configuration: { comment: 12345 } },
    });

    expect(() => createSilenceMapper.props(ctx)).not.toThrow();
  });

  it("does not throw when configuration.comment is an object", () => {
    const ctx = buildComponentContext("grafana.createSilence", {
      node: { configuration: { comment: { reason: "oops" } } },
    });

    expect(() => createSilenceMapper.props(ctx)).not.toThrow();
  });

  it("renders a comment metadata entry when comment is a long string", () => {
    const longComment = "a".repeat(120);
    const ctx = buildComponentContext("grafana.createSilence", {
      node: { configuration: { comment: longComment } },
    });

    const props = createSilenceMapper.props(ctx);
    const commentEntry = (props.metadata ?? []).find((item) => item.icon === "sticky-note");
    expect(commentEntry).toBeDefined();
    expect(typeof commentEntry?.label).toBe("string");
    expect((commentEntry?.label as string).endsWith("...")).toBe(true);
  });
});

describe("incident_shared.truncate", () => {
  it("returns undefined for nullish input", () => {
    expect(truncate(undefined, 10)).toBeUndefined();
    expect(truncate(null, 10)).toBeUndefined();
  });

  it("truncates a long string with ellipsis", () => {
    expect(truncate("hello world hello", 5)).toBe("hello...");
  });

  it("returns short strings unchanged", () => {
    expect(truncate("hi", 10)).toBe("hi");
  });

  it("coerces a non-string value to string before truncating", () => {
    expect(truncate(123456789, 4)).toBe("1234...");
  });

  it("does not throw when given an object value", () => {
    expect(() => truncate({ foo: "bar" }, 5)).not.toThrow();
  });
});
