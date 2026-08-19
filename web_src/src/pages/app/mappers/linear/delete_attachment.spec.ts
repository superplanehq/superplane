import { describe, expect, it } from "vitest";

import { deleteAttachmentMapper } from "./delete_attachment";
import { eventStateRegistry } from "./index";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "linear.deleteAttachment",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "linear.attachment",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date("2026-03-26T19:29:35Z").toISOString(),
    updatedAt: new Date("2026-03-26T19:29:35Z").toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  };
}

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

function buildSubtitleCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): SubtitleContext {
  return {
    node: buildNode(overrides?.node),
    execution: buildExecution(overrides?.execution),
  };
}

function buildComponentContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastExecutions?: ExecutionInfo[];
  componentDefinition?: Partial<ComponentDefinition>;
}): ComponentBaseContext {
  const node = buildNode(overrides?.node);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "linear.deleteAttachment",
      label: "Delete Attachment",
      description: "",
      icon: "linear",
      color: "indigo",
      ...overrides?.componentDefinition,
    },
    lastExecutions: overrides?.lastExecutions ?? [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("deleteAttachmentMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteAttachmentMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("reports the deleted attachment", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ id: "a7c41e88", deleted: true })] } },
    });
    const details = deleteAttachmentMapper.getExecutionDetails(ctx);

    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Attachment"]).toBe("a7c41e88");
    expect(details["Deleted"]).toBe("Yes");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });
});

describe("deleteAttachmentMapper.subtitle", () => {
  it("does not throw without outputs", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: undefined } });
    expect(() => deleteAttachmentMapper.subtitle!(ctx)).not.toThrow();
  });
});

describe("deleteAttachmentMapper.props", () => {
  it("surfaces the configured issue and attachment", () => {
    const props = deleteAttachmentMapper.props(
      buildComponentContext({
        node: { configuration: { issue: "ENG-142", attachment: "a7c41e88" } },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "hash", label: "ENG-142" });
    expect(props.metadata?.[1]).toEqual({ icon: "link", label: "a7c41e88" });
  });

  it("shows only the attachment when the issue is left empty", () => {
    const props = deleteAttachmentMapper.props(
      buildComponentContext({ node: { configuration: { attachment: "a7c41e88" } } }),
    );

    expect(props.metadata).toHaveLength(1);
    expect(props.metadata?.[0]).toEqual({ icon: "link", label: "a7c41e88" });
  });
});

describe("eventStateRegistry", () => {
  it("registers deleteAttachment", () => {
    expect(eventStateRegistry.deleteAttachment).toBeDefined();
  });
});
