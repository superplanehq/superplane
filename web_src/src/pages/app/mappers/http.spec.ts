import { describe, expect, it } from "vitest";

import { httpMapper } from "./http";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "./types";

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

type TestExecution = Omit<ExecutionInfo, "updatedAt"> & { updatedAt?: string };

function buildExecution({
  outputs,
  state = "STATE_FINISHED",
  result = "RESULT_PASSED",
  resultReason = "RESULT_REASON_OK",
  resultMessage = "",
  updatedAt,
  configuration,
  createdAt,
}: {
  outputs?: { success?: OutputPayload[]; failure?: OutputPayload[] };
  state?: ExecutionInfo["state"];
  result?: ExecutionInfo["result"];
  resultReason?: ExecutionInfo["resultReason"];
  resultMessage?: string;
  updatedAt?: string;
  configuration?: Record<string, unknown>;
  createdAt?: string;
}) {
  const now = createdAt ?? new Date().toISOString();

  const execution: TestExecution = {
    id: "exec-1",
    createdAt: now,
    state,
    result,
    resultReason,
    resultMessage,
    metadata: {},
    configuration: configuration ?? {},
    rootEvent: undefined,
    outputs,
  };

  if (updatedAt !== undefined) {
    execution.updatedAt = updatedAt;
  }

  return execution as ExecutionInfo;
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

    expect(httpMapper.getExecutionDetails(ctx)).toMatchObject({ Response: "200" });
  });

  it("includes Request from execution configuration", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          success: [{ type: "json", timestamp: new Date().toISOString(), data: { status: 200 } }],
        },
        configuration: { method: "POST", url: "https://example.com/api" },
      }),
    };

    const details = httpMapper.getExecutionDetails(ctx);
    expect(details["Request"]).toBe("POST https://example.com/api");
    expect(details["Response"]).toBe("200");
  });

  it("does not include Request when configuration is missing url", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          success: [{ type: "json", timestamp: new Date().toISOString(), data: { status: 200 } }],
        },
        configuration: { method: "GET" },
      }),
    };

    const details = httpMapper.getExecutionDetails(ctx);
    expect(details["Request"]).toBeUndefined();
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
