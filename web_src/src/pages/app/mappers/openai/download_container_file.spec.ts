import { describe, expect, it } from "vitest";

import { downloadContainerFileMapper } from "./download_container_file";
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
  name: "openai.downloadContainerFile",
  label: "Download Container File",
  description: "",
  icon: "file-down",
  color: "#6B7280",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "openai.downloadContainerFile",
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
  return { type: "openai.containerFile.downloaded", timestamp: new Date().toISOString(), data };
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

describe("downloadContainerFileMapper.getExecutionDetails", () => {
  it("does not throw when outputs are missing", () => {
    expect(() =>
      downloadContainerFileMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: undefined } })),
    ).not.toThrow();
    expect(() =>
      downloadContainerFileMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: { default: [] } } })),
    ).not.toThrow();
  });

  it("surfaces executed-at first, then the container file details", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              fileId: "cfile_682e0e",
              containerId: "cntr_682e0e",
              path: "/mnt/data/plot.png",
              filename: "plot.png",
              bytes: 880,
              encoding: "base64",
              content: "iVBORw0...",
            }),
          ],
        },
      },
    });
    const details = downloadContainerFileMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Filename"]).toBe("plot.png");
    expect(details["Path"]).toBe("/mnt/data/plot.png");
    expect(details["Size"]).toBe("880 B");
    expect(details["Container ID"]).toBe("cntr_682e0e");
    expect(details["File ID"]).toBe("cfile_682e0e");
    expect(details["Content"]).toBeUndefined();
    expect(Object.keys(details)).toHaveLength(6);
  });

  it("skips empty values", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ fileId: "cfile_682e0e" })] } },
    });
    const details = downloadContainerFileMapper.getExecutionDetails(ctx);
    expect(details["File ID"]).toBe("cfile_682e0e");
    expect(details["Filename"]).toBeUndefined();
    expect(details["Path"]).toBeUndefined();
    expect(details["Size"]).toBeUndefined();
    expect(details["Container ID"]).toBeUndefined();
  });
});

describe("downloadContainerFileMapper.props", () => {
  it("shows literal container and file ids from the configuration", () => {
    const props = downloadContainerFileMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { containerId: "cntr_682e0e", fileId: "cfile_682e0e" } }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "container", label: "cntr_682e0e" },
      { icon: "file-text", label: "cfile_682e0e" },
    ]);
  });

  it("skips configuration values that are expressions", () => {
    const props = downloadContainerFileMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: {
            containerId: "{{ inputs.default.data.artifacts[0].containerId }}",
            fileId: "cfile_682e0e",
          },
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "file-text", label: "cfile_682e0e" }]);
  });
});

describe("eventStateRegistry.downloadContainerFile", () => {
  it("maps finished passed to downloaded", () => {
    expect(eventStateRegistry.downloadContainerFile.getState(buildExecution())).toBe("downloaded");
  });
});
