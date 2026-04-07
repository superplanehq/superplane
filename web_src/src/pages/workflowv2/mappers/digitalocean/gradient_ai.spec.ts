import { describe, expect, it } from "vitest";

import { createKnowledgeBaseMapper } from "./create_knowledge_base";
import { attachKnowledgeBaseMapper } from "./attach_knowledge_base";
import { deleteKnowledgeBaseMapper } from "./delete_knowledge_base";
import { runEvaluationMapper, RUN_EVALUATION_STATE_REGISTRY } from "./run_evaluation";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "digitalocean.createKnowledgeBase",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "digitalocean.result",
    timestamp: new Date().toISOString(),
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

// ── createKnowledgeBaseMapper ────────────────────────────────────────

describe("createKnowledgeBaseMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data fields are all missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => createKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when node configuration and metadata are undefined", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: undefined, metadata: undefined },
      execution: { outputs: { default: [buildOutput({ name: "kb" })] } },
    });
    expect(() => createKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts knowledge base details from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "my-kb",
              uuid: "kb-uuid",
              databaseId: "db-1",
              region: "nyc3",
              embeddingModelName: "text-embedding-3",
              projectName: "proj",
              tags: ["a", "b"],
            }),
          ],
        },
      },
    });
    const details = createKnowledgeBaseMapper.getExecutionDetails(ctx);
    expect(details["Knowledge Base"]).toBe("my-kb");
    expect(details["Region"]).toBe("nyc3");
    expect(details["Embedding Model"]).toBe("text-embedding-3");
    expect(details["Project"]).toBe("proj");
    expect(details["Tags"]).toBe("a, b");
    expect(details["View Knowledge Base"]).toContain("kb-uuid");
    expect(details["View OpenSearch Database"]).toContain("db-1");
  });

  it("omits links when uuid and databaseId are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "kb" })] } },
    });
    const details = createKnowledgeBaseMapper.getExecutionDetails(ctx);
    expect(details["View Knowledge Base"]).toBeUndefined();
    expect(details["View OpenSearch Database"]).toBeUndefined();
  });

  it("omits tags when the array is empty", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "kb", tags: [] })] } },
    });
    expect(createKnowledgeBaseMapper.getExecutionDetails(ctx)["Tags"]).toBeUndefined();
  });
});

// ── attachKnowledgeBaseMapper ────────────────────────────────────────

describe("attachKnowledgeBaseMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => attachKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => attachKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("prefers node metadata names over output IDs", () => {
    const ctx = buildDetailsCtx({
      node: { metadata: { agentName: "My Agent", knowledgeBaseName: "My KB" } },
      execution: { outputs: { default: [buildOutput({ agentUUID: "a1", knowledgeBaseUUID: "kb1" })] } },
    });
    const details = attachKnowledgeBaseMapper.getExecutionDetails(ctx);
    expect(details["Agent"]).toBe("My Agent");
    expect(details["Knowledge Base"]).toBe("My KB");
  });

  it("falls back to output IDs when metadata is absent", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ agentUUID: "agent-1", knowledgeBaseUUID: "kb-2" })] } },
    });
    const details = attachKnowledgeBaseMapper.getExecutionDetails(ctx);
    expect(details["Agent"]).toBe("agent-1");
    expect(details["Knowledge Base"]).toBe("kb-2");
  });

  it("shows dash when both metadata and output IDs are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = attachKnowledgeBaseMapper.getExecutionDetails(ctx);
    expect(details["Agent"]).toBe("-");
    expect(details["Knowledge Base"]).toBe("-");
  });

  it("does not throw when node metadata is undefined", () => {
    const ctx = buildDetailsCtx({
      node: { metadata: undefined },
      execution: { outputs: { default: [buildOutput({ agentUUID: "a1" })] } },
    });
    expect(() => attachKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

// ── deleteKnowledgeBaseMapper ────────────────────────────────────────

describe("deleteKnowledgeBaseMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deleteKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("uses node metadata name for knowledge base when available", () => {
    const ctx = buildDetailsCtx({
      node: { metadata: { knowledgeBaseName: "My KB" } },
      execution: {
        outputs: { default: [buildOutput({ knowledgeBaseUUID: "kb-1", databaseDeleted: true, databaseName: "db-1" })] },
      },
    });
    const details = deleteKnowledgeBaseMapper.getExecutionDetails(ctx);
    expect(details["Knowledge Base"]).toBe("My KB");
  });

  it("falls back to output knowledgeBaseId when metadata is absent", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ knowledgeBaseUUID: "kb-99", databaseDeleted: false })] },
      },
    });
    expect(deleteKnowledgeBaseMapper.getExecutionDetails(ctx)["Knowledge Base"]).toBe("kb-99");
  });

  it("shows database deleted status with name", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ knowledgeBaseUUID: "kb-1", databaseDeleted: true, databaseName: "db-1" })] },
      },
    });
    expect(deleteKnowledgeBaseMapper.getExecutionDetails(ctx)["OpenSearch Database"]).toContain("Deleted");
    expect(deleteKnowledgeBaseMapper.getExecutionDetails(ctx)["OpenSearch Database"]).toContain("db-1");
  });

  it("shows database kept when not deleted", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ knowledgeBaseUUID: "kb-1", databaseDeleted: false })] } },
    });
    expect(deleteKnowledgeBaseMapper.getExecutionDetails(ctx)["OpenSearch Database"]).toBe("Kept");
  });

  it("does not throw when node metadata and configuration are undefined", () => {
    const ctx = buildDetailsCtx({
      node: { metadata: undefined, configuration: undefined },
      execution: { outputs: { default: [buildOutput({ knowledgeBaseUUID: "kb-1", databaseDeleted: false })] } },
    });
    expect(() => deleteKnowledgeBaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

// ── runEvaluationMapper ──────────────────────────────────────────────

describe("runEvaluationMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => runEvaluationMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when passed and failed are both empty arrays", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { passed: [], failed: [] } } });
    expect(() => runEvaluationMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data fields are all missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { passed: [buildOutput({})] } } });
    expect(() => runEvaluationMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when node metadata and configuration are undefined", () => {
    const ctx = buildDetailsCtx({
      node: { metadata: undefined, configuration: undefined },
      execution: { outputs: { passed: [buildOutput({ testCaseName: "T" })] } },
    });
    expect(() => runEvaluationMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts details from a passed evaluation", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          passed: [
            buildOutput({
              testCaseName: "My Test",
              finishedAt: new Date().toISOString(),
              workspaceUUID: "ws-1",
              testCaseUUID: "tc-1",
              evaluationRunUUID: "run-1",
              starMetric: { metricName: "Accuracy", numberValue: 92.567 },
              prompts: [{}, {}],
            }),
          ],
        },
      },
    });
    const details = runEvaluationMapper.getExecutionDetails(ctx);
    expect(details["Test Case"]).toBe("My Test");
    expect(details["Star Metric"]).toContain("Accuracy");
    expect(details["Prompts Evaluated"]).toBe("2");
    expect(details["View Evaluation"]).toContain("ws-1");
    expect(details["Finished At"]).toBeDefined();
  });

  it("extracts details from a failed evaluation", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { failed: [buildOutput({ testCaseName: "Bad Test", errorDescription: "timeout" })] },
      },
    });
    const details = runEvaluationMapper.getExecutionDetails(ctx);
    expect(details["Test Case"]).toBe("Bad Test");
    expect(details["Error"]).toBe("timeout");
  });

  it("prefers passed output over failed when both are present", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          passed: [buildOutput({ testCaseName: "From Passed" })],
          failed: [buildOutput({ testCaseName: "From Failed" })],
        },
      },
    });
    expect(runEvaluationMapper.getExecutionDetails(ctx)["Test Case"]).toBe("From Passed");
  });

  it("omits star metric when metricName is missing", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { passed: [buildOutput({ testCaseName: "T", starMetric: { numberValue: 50 } })] },
      },
    });
    expect(runEvaluationMapper.getExecutionDetails(ctx)["Star Metric"]).toBeUndefined();
  });

  it("omits view evaluation link when any required ID is missing", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { passed: [buildOutput({ testCaseName: "T", workspaceUUID: "ws-1" })] },
      },
    });
    expect(runEvaluationMapper.getExecutionDetails(ctx)["View Evaluation"]).toBeUndefined();
  });
});

// ── RUN_EVALUATION_STATE_REGISTRY ────────────────────────────────────

describe("RUN_EVALUATION_STATE_REGISTRY", () => {
  it("returns 'passed' when passed outputs exist", () => {
    const execution = buildExecution({ outputs: { passed: [buildOutput({})] } });
    expect(RUN_EVALUATION_STATE_REGISTRY.getState(execution)).toBe("passed");
  });

  it("returns 'failed' when only failed outputs exist", () => {
    const execution = buildExecution({ outputs: { failed: [buildOutput({})] } });
    expect(RUN_EVALUATION_STATE_REGISTRY.getState(execution)).toBe("failed");
  });

  it("returns 'success' when both output buckets are empty", () => {
    const execution = buildExecution({ outputs: { passed: [], failed: [] } });
    expect(RUN_EVALUATION_STATE_REGISTRY.getState(execution)).toBe("success");
  });

  it("returns 'success' when outputs is undefined", () => {
    const execution = buildExecution({ outputs: undefined });
    expect(RUN_EVALUATION_STATE_REGISTRY.getState(execution)).toBe("success");
  });

  it("returns running state when execution is still in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(RUN_EVALUATION_STATE_REGISTRY.getState(execution)).toBe("running");
  });

  it("returns error state when execution failed with error reason", () => {
    const execution = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
      resultReason: "RESULT_REASON_ERROR",
      resultMessage: "something went wrong",
    });
    expect(RUN_EVALUATION_STATE_REGISTRY.getState(execution)).toBe("error");
  });
});
