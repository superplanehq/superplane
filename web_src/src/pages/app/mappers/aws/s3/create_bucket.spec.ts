import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../../types";
import { createBucketMapper } from "./create_bucket";
import { eventStateRegistry } from "../index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Bucket Node",
    componentName: "aws.s3.createBucket",
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
  name: "aws.s3.createBucket",
  label: "S3 • Create Bucket",
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
  region: "us-east-1",
  arn: "arn:aws:s3:::my-example-bucket",
};

describe("createBucketMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createBucketMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts bucket fields from output", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(bucketOutput)] } } });
    const details = createBucketMapper.getExecutionDetails(ctx);
    expect(details["Bucket"]).toBe("my-example-bucket");
    expect(details["Region"]).toBe("us-east-1");
    expect(details["ARN"]).toBe("arn:aws:s3:::my-example-bucket");
  });
});

describe("createBucketMapper.props", () => {
  it("uses node name as title", () => {
    expect(createBucketMapper.props(buildPropsContext()).title).toBe("Create Bucket Node");
  });

  it("includes bucket name from configuration in metadata", () => {
    const props = createBucketMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { bucketName: "my-example-bucket", region: "us-east-1" } }) }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("my-example-bucket");
  });
});

describe("eventStateRegistry['s3.createBucket']", () => {
  it("maps finished success to created", () => {
    expect(eventStateRegistry["s3.createBucket"].getState(buildExecution())).toBe("created");
  });
});
