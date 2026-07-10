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
  name: "claude.downloadFile",
  label: "Download File",
  description: "",
  icon: "file-down",
  color: "#C9784D",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "claude.downloadFile",
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
  return { type: "claude.file.downloaded", timestamp: new Date().toISOString(), data };
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
              id: "file_011CNha8iCJcU1wXNR6q4V8w",
              filename: "chart.png",
              mimeType: "image/png",
              sizeBytes: 34567,
              encoding: "base64",
              content: "iVBORw0KGgo...",
            }),
          ],
        },
      },
    });
    const details = downloadFileMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Filename"]).toBe("chart.png");
    expect(details["MIME Type"]).toBe("image/png");
    expect(details["Size"]).toBe("33.8 KB");
    expect(details["Encoding"]).toBe("base64");
    expect(details["Content"]).toBeUndefined();
    expect(Object.keys(details)).toHaveLength(5);
  });

  it("skips empty values", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ filename: "notes.txt" })] } },
    });
    const details = downloadFileMapper.getExecutionDetails(ctx);
    expect(details["Filename"]).toBe("notes.txt");
    expect(details["MIME Type"]).toBeUndefined();
    expect(details["Size"]).toBeUndefined();
    expect(details["Encoding"]).toBeUndefined();
  });
});

describe("downloadFileMapper.props", () => {
  it("shows the filename from node metadata", () => {
    const props = downloadFileMapper.props(
      buildPropsContext({ node: buildNode({ metadata: { filename: "chart.png" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "file-text", label: "chart.png" }]);
  });

  it("falls back to the configured file when metadata is absent", () => {
    const props = downloadFileMapper.props(
      buildPropsContext({ node: buildNode({ metadata: {}, configuration: { file: "file_011" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "file-text", label: "file_011" }]);
  });
});

describe("eventStateRegistry.downloadFile", () => {
  it("maps finished passed to downloaded", () => {
    expect(eventStateRegistry.downloadFile.getState(buildExecution())).toBe("downloaded");
  });
});
