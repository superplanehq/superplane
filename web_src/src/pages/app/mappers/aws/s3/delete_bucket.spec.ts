import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../../types";
import { deleteBucketMapper } from "./delete_bucket";
import { eventStateRegistry } from "../index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Delete Bucket Node",
    componentName: "aws.s3.deleteBucket",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return { type: "aws.s3.bucket.deleted", timestamp: new Date().toISOString(), data };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: "2026-06-01T10:00:00.000Z",
    updatedAt: "2026-06-01T10:00:05.000Z",
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

function buildDetailsCtx(overrides?: { node?: Partial<NodeInfo>; execution?: Partial<ExecutionInfo> }): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const defaultDefinition: ComponentDefinition = {
  name: "aws.s3.deleteBucket",
  label: "S3 • Delete Bucket",
  description: "",
  icon: "aws",
  color: "gray",
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

const deletedOutput = { bucketName: "my-example-bucket", deleted: true };

describe("deleteBucketMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteBucketMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts deletion fields from output", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(deletedOutput)] } } });
    const details = deleteBucketMapper.getExecutionDetails(ctx);
    expect(details["Bucket"]).toBe("my-example-bucket");
    expect(details["Deleted"]).toBe("Yes");
  });
});

describe("deleteBucketMapper.props", () => {
  it("uses node name as title", () => {
    expect(deleteBucketMapper.props(buildPropsContext()).title).toBe("Delete Bucket Node");
  });

  it("includes bucket from configuration in metadata", () => {
    const props = deleteBucketMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { bucket: "my-example-bucket", region: "us-east-1" } }) }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("my-example-bucket");
  });
});

describe("eventStateRegistry['s3.deleteBucket']", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry["s3.deleteBucket"].getState(buildExecution())).toBe("deleted");
  });
});
