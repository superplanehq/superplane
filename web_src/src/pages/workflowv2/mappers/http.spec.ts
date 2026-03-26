import { describe, expect, it } from "vitest";

import { httpMapper } from "./http";
import type { ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "./types";

function buildNode(): NodeInfo {
  return {
    id: "node-1",
    name: "HTTP",
    componentName: "http",
    isCollapsed: false,
    configuration: {},
    metadata: {},
  };
}

function buildExecution({
  outputs,
  state = "STATE_FINISHED",
  result = "RESULT_SUCCEEDED",
  resultReason = "RESULT_REASON_UNSPECIFIED",
  resultMessage = "",
  updatedAt,
}: {
  outputs?: { success?: OutputPayload[]; failure?: OutputPayload[] };
  state?: string;
  result?: string;
  resultReason?: string;
  resultMessage?: string;
  updatedAt?: string;
}) {
  const now = new Date().toISOString();

  const execution: any = {
    id: "exec-1",
    createdAt: now,
    state,
    result,
    resultReason,
    resultMessage,
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    outputs,
  };

  if (updatedAt !== undefined) {
    execution.updatedAt = updatedAt;
  }

  return execution;
}

describe("httpMapper.getExecutionDetails", () => {
  it("does not throw when success response.status is undefined", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          success: [{ type: "json", timestamp: new Date().toISOString(), data: { status: undefined } }],
        },
      }),
    };

    expect(() => httpMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(httpMapper.getExecutionDetails(ctx)).toEqual({});
  });

  it("does not throw when failure response.status is undefined", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          failure: [{ type: "json", timestamp: new Date().toISOString(), data: { status: undefined } }],
        },
        result: "RESULT_FAILED",
      }),
    };

    expect(() => httpMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(httpMapper.getExecutionDetails(ctx)).toEqual({});
  });

  it("does not throw when response.status is null", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          success: [{ type: "json", timestamp: new Date().toISOString(), data: { status: null } }],
        },
      }),
    };

    expect(() => httpMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(httpMapper.getExecutionDetails(ctx)).toEqual({});
  });

  it("does not throw when outputs.success is an empty array", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          success: [],
        },
      }),
    };

    expect(() => httpMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(httpMapper.getExecutionDetails(ctx)).toEqual({});
  });

  it("falls through to failure when success is present but empty", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          success: [],
          failure: [{ type: "json", timestamp: new Date().toISOString(), data: { status: 500 } }],
        },
        result: "RESULT_FAILED",
      }),
    };

    expect(httpMapper.getExecutionDetails(ctx)).toEqual({ Response: "500" });
  });

  it("includes Response when status is a number", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          success: [{ type: "json", timestamp: new Date().toISOString(), data: { status: 200 } }],
        },
      }),
    };

    expect(httpMapper.getExecutionDetails(ctx)).toEqual({ Response: "200" });
  });
});

describe("httpMapper.subtitle", () => {
  it("does not throw when outputs.success is empty and returns empty string when there's no timestamp to render", () => {
    const node = buildNode();
    const ctx: SubtitleContext = {
      node,
      execution: buildExecution({
        outputs: { success: [] },
        updatedAt: undefined,
      }),
    };

    expect(() => httpMapper.subtitle(ctx)).not.toThrow();
    expect(httpMapper.subtitle(ctx)).toBe("");
  });

  it("returns a string response line when status is present and updatedAt is missing", () => {
    const node = buildNode();
    const ctx: SubtitleContext = {
      node,
      execution: buildExecution({
        outputs: { success: [{ type: "json", timestamp: new Date().toISOString(), data: { status: 201 } }] },
        updatedAt: undefined,
      }),
    };

    expect(httpMapper.subtitle(ctx)).toBe("Response: 201");
  });

  it("falls through to failure when success is present but empty", () => {
    const node = buildNode();
    const ctx: SubtitleContext = {
      node,
      execution: buildExecution({
        outputs: {
          success: [],
          failure: [{ type: "json", timestamp: new Date().toISOString(), data: { status: 500 } }],
        },
        result: "RESULT_FAILED",
        updatedAt: undefined,
      }),
    };

    expect(httpMapper.subtitle(ctx)).toBe("Response: 500");
  });
});
