import { describe, expect, it } from "vitest";

import { getFileMapper } from "./get_file";
import { eventStateRegistry } from "./index";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";

const defaultDefinition: ComponentDefinition = {
  name: "openai.getFile",
  label: "Get File",
  description: "",
  icon: "file-text",
  color: "#6B7280",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "openai.getFile",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
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

function buildOutput(data: unknown): OutputPayload {
  return { type: "openai.file.fetched", timestamp: new Date().toISOString(), data };
}

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

describe("getFileMapper.getExecutionDetails", () => {
  it("does not throw when outputs are missing", () => {
    expect(() =>
      getFileMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: undefined } })),
    ).not.toThrow();
    expect(() =>
      getFileMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: { default: [] } } })),
    ).not.toThrow();
  });

  it("surfaces executed-at first, then the file metadata", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              id: "file-abc123",
              filename: "salesOverview.pdf",
              purpose: "assistants",
              bytes: 175,
              createdAt: "2026-02-13T12:00:00Z",
              expiresAt: "2026-03-13T12:00:00Z",
              url: "https://platform.openai.com/storage/files/file-abc123",
            }),
          ],
        },
      },
    });
    const details = getFileMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Filename"]).toBe("salesOverview.pdf");
    expect(details["Purpose"]).toBe("assistants");
    expect(details["Size"]).toBe("175 B");
    expect(details["File ID"]).toBe("file-abc123");
    expect(details["Link"]).toBe("https://platform.openai.com/storage/files/file-abc123");
    expect(Object.keys(details)).toHaveLength(6);
  });

  it("skips empty values", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ id: "file-abc123" })] } },
    });
    const details = getFileMapper.getExecutionDetails(ctx);
    expect(details["File ID"]).toBe("file-abc123");
    expect(details["Filename"]).toBeUndefined();
    expect(details["Purpose"]).toBeUndefined();
    expect(details["Size"]).toBeUndefined();
    expect(details["Link"]).toBeUndefined();
  });
});

describe("getFileMapper.props", () => {
  it("shows the filename from node metadata", () => {
    const props = getFileMapper.props(
      buildPropsContext({ node: buildNode({ metadata: { filename: "salesOverview.pdf" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "file-text", label: "salesOverview.pdf" }]);
  });

  it("falls back to the configured file when metadata is absent", () => {
    const props = getFileMapper.props(
      buildPropsContext({ node: buildNode({ metadata: {}, configuration: { file: "file-abc123" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "file-text", label: "file-abc123" }]);
  });
});

describe("eventStateRegistry.getFile", () => {
  it("maps finished passed to fetched", () => {
    expect(eventStateRegistry.getFile.getState(buildExecution())).toBe("fetched");
  });
});
