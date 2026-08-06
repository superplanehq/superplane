import { describe, expect, it } from "vitest";

import { createAttachmentMapper } from "./create_attachment";
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
    componentName: "linear.createAttachment",
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
      name: "linear.createAttachment",
      label: "Create Attachment",
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

const attachmentPayload = {
  id: "a7c41e88",
  title: "Deploy - staging",
  subtitle: "Succeeded in 4m 12s",
  url: "https://app.superplane.com/runs/8f21c4d0",
  createdAt: "2026-03-27T16:42:11.123Z",
  issue: {
    id: "i1",
    identifier: "ENG-142",
    title: "Deploy pipeline fails on retry",
    url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry",
  },
};

describe("createAttachmentMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createAttachmentMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("always includes Executed At first", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(attachmentPayload)] } } });
    const details = createAttachmentMapper.getExecutionDetails(ctx);

    expect(Object.keys(details)[0]).toBe("Executed At");
  });

  it("extracts the attachment fields that matter, including the link", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(attachmentPayload)] } } });
    const details = createAttachmentMapper.getExecutionDetails(ctx);

    expect(details["Attachment URL"]).toBe("https://app.superplane.com/runs/8f21c4d0");
    expect(details["Issue"]).toBe("ENG-142 · Deploy pipeline fails on retry");
    expect(details["Title"]).toBe("Deploy - staging");
    expect(details["Subtitle"]).toBe("Succeeded in 4m 12s");
  });

  it("shows at most six details", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(attachmentPayload)] } } });
    const details = createAttachmentMapper.getExecutionDetails(ctx);

    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });

  it("omits rows the payload does not carry", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ id: "a1", title: "Deploy" })] } },
    });
    const details = createAttachmentMapper.getExecutionDetails(ctx);

    expect(details["Subtitle"]).toBeUndefined();
    expect(details["Issue"]).toBeUndefined();
  });
});

describe("createAttachmentMapper.subtitle", () => {
  it("uses the attachment title when present", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: { default: [buildOutput(attachmentPayload)] } } });
    expect(createAttachmentMapper.subtitle!(ctx)).toBe("Deploy - staging");
  });

  it("does not throw without outputs", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: undefined } });
    expect(() => createAttachmentMapper.subtitle!(ctx)).not.toThrow();
  });
});

describe("createAttachmentMapper.props", () => {
  it("surfaces the configured issue and title", () => {
    const props = createAttachmentMapper.props(
      buildComponentContext({
        node: { configuration: { issue: "ENG-142", title: "Deploy - staging" } },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "hash", label: "ENG-142" });
    expect(props.metadata?.[1]).toEqual({ icon: "link", label: "Deploy - staging" });
  });

  it("does not throw when configuration is empty", () => {
    expect(() => createAttachmentMapper.props(buildComponentContext())).not.toThrow();
  });
});

describe("eventStateRegistry", () => {
  it("registers createAttachment", () => {
    expect(eventStateRegistry.createAttachment).toBeDefined();
  });
});
