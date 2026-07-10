import { describe, expect, it } from "vitest";

import { downloadFileMapper } from "./download_file";
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
  name: "openai.downloadFile",
  label: "Download File",
  description: "",
  icon: "file-down",
  color: "#6B7280",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "openai.downloadFile",
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
  return { type: "openai.file.downloaded", timestamp: new Date().toISOString(), data };
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

describe("downloadFileMapper.getExecutionDetails", () => {
  it("does not throw when outputs are missing", () => {
    expect(() =>
      downloadFileMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: undefined } })),
    ).not.toThrow();
    expect(() =>
      downloadFileMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: { default: [] } } })),
    ).not.toThrow();
  });

  it("surfaces executed-at first, then the downloaded file details", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              id: "file-abc123",
              filename: "results.csv",
              purpose: "batch_output",
              bytes: 880,
              encoding: "text",
              content: "a,b\n1,2\n",
              url: "https://platform.openai.com/storage/files/file-abc123",
            }),
          ],
        },
      },
    });
    const details = downloadFileMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Filename"]).toBe("results.csv");
    expect(details["Purpose"]).toBe("batch_output");
    expect(details["Size"]).toBe("880 B");
    expect(details["Encoding"]).toBe("text");
    expect(details["Link"]).toBe("https://platform.openai.com/storage/files/file-abc123");
    expect(details["Content"]).toBeUndefined();
    expect(Object.keys(details)).toHaveLength(6);
  });

  it("skips empty values", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ filename: "results.csv" })] } },
    });
    const details = downloadFileMapper.getExecutionDetails(ctx);
    expect(details["Filename"]).toBe("results.csv");
    expect(details["Purpose"]).toBeUndefined();
    expect(details["Size"]).toBeUndefined();
    expect(details["Encoding"]).toBeUndefined();
    expect(details["Link"]).toBeUndefined();
  });
});

describe("downloadFileMapper.props", () => {
  it("shows the filename from node metadata", () => {
    const props = downloadFileMapper.props(
      buildPropsContext({ node: buildNode({ metadata: { filename: "results.csv" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "file-text", label: "results.csv" }]);
  });

  it("falls back to the configured file when metadata is absent", () => {
    const props = downloadFileMapper.props(
      buildPropsContext({ node: buildNode({ metadata: {}, configuration: { file: "file-abc123" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "file-text", label: "file-abc123" }]);
  });
});

describe("eventStateRegistry.downloadFile", () => {
  it("maps finished passed to downloaded", () => {
    expect(eventStateRegistry.downloadFile.getState(buildExecution())).toBe("downloaded");
  });
});
