import { describe, expect, it } from "vitest";

import { createKnowledgeBaseMapper } from "./create_knowledge_base";
import { getKnowledgeBaseMapper } from "./get_knowledge_base";
import { indexKnowledgeBaseMapper } from "./index_knowledge_base";
import { addDataSourceMapper } from "./add_data_source";
import { deleteDataSourceMapper } from "./delete_data_source";
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
});

// ── getKnowledgeBaseMapper ───────────────────────────────────────────

describe("getKnowledgeBaseMapper.getExecutionDetails", () => {
  it("extracts knowledge base details from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              uuid: "kb-uuid",
              // name: "my-kb",
              databaseStatus: "ONLINE",
              database: { id: "db-1", name: "my-kb-os" },
              dataSources: [
                { uuid: "ds-1", type: "spaces" },
                { uuid: "ds-2", type: "web" },
              ],
              lastIndexingJob: {
                status: "INDEX_JOB_STATUS_COMPLETED",
                completedDataSources: 2,
                totalDataSources: 2,
                finishedAt: "2025-06-01T00:05:32Z",
              },
            }),
          ],
        },
      },
    });
    const details = getKnowledgeBaseMapper.getExecutionDetails(ctx);
    // expect(details["Knowledge Base"]).toBe("my-kb");
    expect(details["View Knowledge Base"]).toContain("kb-uuid");
    expect(details["Knowledge Base Endpoint"]).toContain("kbaas.do-ai.run/v1/kb-uuid/retrieve");
    expect(details["Data Sources"]).toBe("2");
    expect(details["Database"]).toBe("my-kb-os");
    expect(details["View Database"]).toContain("db-1");
    expect(details["Last Indexing"]).toContain("Completed");
    expect(details["Last Indexing"]).toContain("2/2 sources");
    expect(details["Last Indexed At"]).toBeDefined();
    expect(details["View Activity"]).toContain("kb-uuid/activity");
  });
});

// ── indexKnowledgeBaseMapper ────────────────────────────────────────

describe("indexKnowledgeBaseMapper.getExecutionDetails", () => {
  it("extracts indexing job details from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              knowledgeBaseUUID: "kb-uuid",
              knowledgeBaseName: "my-kb",
              jobUUID: "job-1",
              status: "INDEX_JOB_STATUS_COMPLETED",
              totalTokens: "1500",
              completedDataSources: 2,
              totalDataSources: 2,
              startedAt: "2025-06-01T00:00:00Z",
              finishedAt: "2025-06-01T00:05:32Z",
            }),
          ],
        },
      },
    });
    const details = indexKnowledgeBaseMapper.getExecutionDetails(ctx);
    expect(details["Started At"]).toBeDefined();
    expect(details["Finished At"]).toBeDefined();
    expect(details["Knowledge Base"]).toBe("my-kb");
    expect(details["View Knowledge Base"]).toContain("kb-uuid");
    expect(details["View Activity"]).toContain("kb-uuid/activity");
    expect(details["Indexing Status"]).toBe("Completed");
    expect(details["Data Sources Indexed"]).toBe("2/2 completed");
    expect(details["Total Tokens"]).toBe("1500");
  });
});

// ── addDataSourceMapper ───────────────────────────────────────────

describe("addDataSourceMapper.getExecutionDetails", () => {
  it("extracts details with indexing job", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              dataSourceUUID: "ds-1",
              dataSourceName: "my-bucket (tor1)",
              knowledgeBaseUUID: "kb-uuid",
              knowledgeBaseName: "my-kb",
              indexingJob: {
                status: "INDEX_JOB_STATUS_COMPLETED",
                totalTokens: "1500",
                completedDataSources: 2,
                totalDataSources: 2,
                finishedAt: "2025-06-01T00:05:32Z",
              },
            }),
          ],
        },
      },
    });
    const details = addDataSourceMapper.getExecutionDetails(ctx);
    expect(details["Knowledge Base"]).toBe("my-kb");
    expect(details["View Knowledge Base"]).toContain("kb-uuid");
    expect(details["Data Source"]).toBe("my-bucket (tor1)");
    expect(details["Indexing Status"]).toBe("Completed");
    expect(details["Total Tokens"]).toBe("1500");
    expect(details["Indexing finished at"]).toBeDefined();
    expect(details["View Activity"]).toContain("kb-uuid/activity");
  });

  it("extracts details without indexing job", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              dataSourceUUID: "ds-1",
              dataSourceName: "https://docs.example.com",
              knowledgeBaseUUID: "kb-uuid",
              knowledgeBaseName: "my-kb",
            }),
          ],
        },
      },
    });
    const details = addDataSourceMapper.getExecutionDetails(ctx);
    expect(details["Knowledge Base"]).toBe("my-kb");
    expect(details["Data Source"]).toBe("https://docs.example.com");
    expect(details["Indexing Status"]).toBeUndefined();
    expect(details["View Activity"]).toBeUndefined();
  });
});

// ── deleteDataSourceMapper ───────────────────────────────────────────

describe("deleteDataSourceMapper.getExecutionDetails", () => {
  it("extracts details with indexing job", () => {
    const ctx = buildDetailsCtx({
      node: { metadata: { knowledgeBaseName: "my-kb", dataSourceName: "my-bucket (tor1)" } },
      execution: {
        outputs: {
          default: [
            buildOutput({
              dataSourceUUID: "ds-1",
              knowledgeBaseUUID: "kb-uuid",
              knowledgeBaseName: "my-kb",
              indexingJob: {
                status: "INDEX_JOB_STATUS_COMPLETED",
                totalTokens: "800",
                completedDataSources: 1,
                totalDataSources: 1,
                finishedAt: "2025-06-01T00:03:12Z",
              },
            }),
          ],
        },
      },
    });
    const details = deleteDataSourceMapper.getExecutionDetails(ctx);
    expect(details["Knowledge Base"]).toBe("my-kb");
    expect(details["View Knowledge Base"]).toContain("kb-uuid");
    expect(details["Data Source"]).toBe("my-bucket (tor1)");
    expect(details["Indexing Status"]).toBe("Completed");
    expect(details["Total Tokens"]).toBe("800");
    expect(details["Indexing finished at"]).toBeDefined();
    expect(details["View Activity"]).toContain("kb-uuid/activity");
  });

  it("extracts details without indexing job", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              dataSourceUUID: "ds-1",
              knowledgeBaseUUID: "kb-uuid",
              knowledgeBaseName: "my-kb",
            }),
          ],
        },
      },
    });
    const details = deleteDataSourceMapper.getExecutionDetails(ctx);
    expect(details["Knowledge Base"]).toBe("my-kb");
    expect(details["Data Source"]).toBe("ds-1");
    expect(details["Indexing Status"]).toBeUndefined();
    expect(details["View Activity"]).toBeUndefined();
  });
});

// ── attachKnowledgeBaseMapper ────────────────────────────────────────

describe("attachKnowledgeBaseMapper.getExecutionDetails", () => {
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
});

// ── deleteKnowledgeBaseMapper ────────────────────────────────────────

describe("deleteKnowledgeBaseMapper.getExecutionDetails", () => {
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
});

// ── runEvaluationMapper ──────────────────────────────────────────────

describe("runEvaluationMapper.getExecutionDetails", () => {
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
    expect(details["Evaluation Error"]).toBe("timeout");
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
