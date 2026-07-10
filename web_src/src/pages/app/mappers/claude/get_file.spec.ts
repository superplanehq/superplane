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
  name: "claude.getFile",
  label: "Get File",
  description: "",
  icon: "file-text",
  color: "#C9784D",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "claude.getFile",
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
  return { type: "claude.file.fetched", timestamp: new Date().toISOString(), data };
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
              id: "file_011CNha8iCJcU1wXNR6q4V8w",
              filename: "quarterly-report.pdf",
              mimeType: "application/pdf",
              sizeBytes: 102400,
              createdAt: "2026-04-15T18:37:24.100435Z",
              downloadable: true,
              downloadUrl: "https://api.anthropic.com/v1/files/file_011CNha8iCJcU1wXNR6q4V8w/content",
            }),
          ],
        },
      },
    });
    const details = getFileMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Filename"]).toBe("quarterly-report.pdf");
    expect(details["MIME Type"]).toBe("application/pdf");
    expect(details["Size"]).toBe("100.0 KB");
    expect(details["Downloadable"]).toBe("Yes");
    expect(details["File ID"]).toBe("file_011CNha8iCJcU1wXNR6q4V8w");
    expect(Object.keys(details)).toHaveLength(6);
  });

  it("marks non-downloadable files and skips empty values", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ id: "file_1", downloadable: false })] } },
    });
    const details = getFileMapper.getExecutionDetails(ctx);
    expect(details["Downloadable"]).toBe("No");
    expect(details["Filename"]).toBeUndefined();
    expect(details["MIME Type"]).toBeUndefined();
    expect(details["Size"]).toBeUndefined();
  });
});

describe("getFileMapper.props", () => {
  it("shows the filename from node metadata", () => {
    const props = getFileMapper.props(
      buildPropsContext({ node: buildNode({ metadata: { filename: "quarterly-report.pdf" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "file-text", label: "quarterly-report.pdf" }]);
  });

  it("falls back to the configured file when metadata is absent", () => {
    const props = getFileMapper.props(
      buildPropsContext({ node: buildNode({ metadata: {}, configuration: { file: "file_011" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "file-text", label: "file_011" }]);
  });

  it("shows no metadata when neither is set", () => {
    const props = getFileMapper.props(buildPropsContext());
    expect(props.metadata).toEqual([]);
  });
});

describe("eventStateRegistry.getFile", () => {
  it("maps finished passed to fetched", () => {
    expect(eventStateRegistry.getFile.getState(buildExecution())).toBe("fetched");
  });
});
