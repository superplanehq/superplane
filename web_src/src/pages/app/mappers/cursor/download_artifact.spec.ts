import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { downloadArtifactMapper } from "./download_artifact";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Download Artifact",
    componentName: "cursor.downloadArtifact",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown, timestamp = "2026-04-13T18:45:00.000Z"): OutputPayload {
  return {
    type: "cursor.downloadArtifact.result",
    timestamp,
    data,
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

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const defaultDefinition: ComponentDefinition = {
  name: "cursor.downloadArtifact",
  label: "Download Artifact",
  description: "",
  icon: "download",
  color: "blue",
};

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

describe("downloadArtifactMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => downloadArtifactMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => downloadArtifactMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("shows artifact details with the timestamp first and the download URL included", () => {
    const url = "https://cloud-agent-artifacts.s3.us-east-1.amazonaws.com/artifacts/screenshot.png?X-Amz-Expires=900";
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              agentId: "bc-00000000-0000-0000-0000-000000000001",
              path: "artifacts/screenshot.png",
              url,
              expiresAt: "2026-04-13T19:00:00.000Z",
            }),
          ],
        },
      },
    });

    const details = downloadArtifactMapper.getExecutionDetails(ctx);
    const keys = Object.keys(details);

    expect(keys[0]).toBe("Downloaded At");
    expect(details["Downloaded At"]).toBe(new Date("2026-04-13T18:45:00.000Z").toLocaleString());
    expect(details["Artifact"]).toBe("artifacts/screenshot.png");
    expect(details["Agent"]).toBe("bc-00000000-0000-0000-0000-000000000001");
    expect(details["Download URL"]).toBe(url);
    expect(details["Expires At"]).toBe(new Date("2026-04-13T19:00:00.000Z").toLocaleString());
    expect(keys.length).toBeLessThanOrEqual(6);
  });

  it("falls back to execution createdAt when the payload has no timestamp", () => {
    const createdAt = "2026-04-13T18:00:00.000Z";
    const ctx = buildDetailsCtx({
      execution: {
        createdAt,
        outputs: { default: [{ type: "cursor.downloadArtifact.result", data: {} } as OutputPayload] },
      },
    });

    const details = downloadArtifactMapper.getExecutionDetails(ctx);
    expect(details["Downloaded At"]).toBe(new Date(createdAt).toLocaleString());
  });

  it("omits fields that are missing from the payload", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ path: "artifacts/report.pdf" })] },
      },
    });

    const details = downloadArtifactMapper.getExecutionDetails(ctx);
    expect(details["Artifact"]).toBe("artifacts/report.pdf");
    expect(details["Agent"]).toBeUndefined();
    expect(details["Download URL"]).toBeUndefined();
    expect(details["Expires At"]).toBeUndefined();
  });
});

describe("downloadArtifactMapper.props", () => {
  it("does not throw with minimal context", () => {
    const ctx = buildPropsContext();
    expect(() => downloadArtifactMapper.props!(ctx)).not.toThrow();
  });

  it("uses the node name as title and the cursor icon", () => {
    const props = downloadArtifactMapper.props!(buildPropsContext());
    expect(props.title).toBe("Download Artifact");
    expect(props.iconSlug).toBe("download");
    expect(props.includeEmptyState).toBe(true);
  });
});
