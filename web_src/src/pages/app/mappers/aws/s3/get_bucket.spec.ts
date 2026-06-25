import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../../types";
import { getBucketMapper } from "./get_bucket";
import { eventStateRegistry } from "../index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Get Bucket Node",
    componentName: "aws.s3.getBucket",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return { type: "aws.s3.bucket", timestamp: new Date().toISOString(), data };
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
  name: "aws.s3.getBucket",
  label: "S3 • Get Bucket",
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

const bucketOutput = {
  bucketName: "my-example-bucket",
  region: "eu-west-1",
  arn: "arn:aws:s3:::my-example-bucket",
};

describe("getBucketMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getBucketMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts bucket fields from output", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(bucketOutput)] } } });
    const details = getBucketMapper.getExecutionDetails(ctx);
    expect(details["Bucket"]).toBe("my-example-bucket");
    expect(details["Region"]).toBe("eu-west-1");
    expect(details["ARN"]).toBe("arn:aws:s3:::my-example-bucket");
  });
});

describe("getBucketMapper.props", () => {
  it("uses node name as title", () => {
    expect(getBucketMapper.props(buildPropsContext()).title).toBe("Get Bucket Node");
  });

  it("includes bucket from configuration in metadata", () => {
    const props = getBucketMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { bucket: "my-example-bucket", region: "eu-west-1" } }) }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("my-example-bucket");
  });
});

describe("eventStateRegistry['s3.getBucket']", () => {
  it("maps finished success to retrieved", () => {
    expect(eventStateRegistry["s3.getBucket"].getState(buildExecution())).toBe("retrieved");
  });
});
