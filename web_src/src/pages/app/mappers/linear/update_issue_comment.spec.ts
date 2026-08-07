import { describe, expect, it } from "vitest";

import { updateIssueCommentMapper } from "./update_issue_comment";
import { eventStateRegistry } from "./index";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "linear.updateIssueComment",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "linear.comment",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date("2026-03-27T18:05:47Z").toISOString(),
    updatedAt: new Date("2026-03-27T18:05:47Z").toISOString(),
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

function buildSubtitleCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): SubtitleContext {
  return {
    node: buildNode(overrides?.node),
    execution: buildExecution(overrides?.execution),
  };
}

function buildComponentContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastExecutions?: ExecutionInfo[];
  componentDefinition?: Partial<ComponentDefinition>;
}): ComponentBaseContext {
  const node = buildNode(overrides?.node);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "linear.updateIssueComment",
      label: "Update Issue Comment",
      description: "",
      icon: "linear",
      color: "indigo",
      ...overrides?.componentDefinition,
    },
    lastExecutions: overrides?.lastExecutions ?? [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

const commentPayload = {
  id: "d3f9b0a2",
  body: "Deploy pipeline is green again. Retried 3 times, all passing.",
  url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry#comment-d3f9b0a2",
  createdAt: "2026-03-27T16:42:11.123Z",
  updatedAt: "2026-03-27T18:05:47.482Z",
  editedAt: "2026-03-27T18:05:47.482Z",
  user: { id: "u1", name: "Ada Lovelace", displayName: "ada" },
  issue: {
    id: "i1",
    identifier: "ENG-142",
    title: "Deploy pipeline fails on retry",
    url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry",
  },
};

describe("updateIssueCommentMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => updateIssueCommentMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("always includes Executed At first", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(commentPayload)] } } });
    const details = updateIssueCommentMapper.getExecutionDetails(ctx);

    expect(details["Executed At"]).toBeDefined();
    expect(Object.keys(details)[0]).toBe("Executed At");
  });

  it("extracts the comment fields that matter, including the link", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(commentPayload)] } } });
    const details = updateIssueCommentMapper.getExecutionDetails(ctx);

    expect(details["Comment URL"]).toBe(
      "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry#comment-d3f9b0a2",
    );
    expect(details["Issue"]).toBe("ENG-142 · Deploy pipeline fails on retry");
    expect(details["Author"]).toBe("ada");
    expect(details["Comment"]).toBe("Deploy pipeline is green again. Retried 3 times, all passing.");
  });

  it("shows when the comment was edited", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(commentPayload)] } } });
    const details = updateIssueCommentMapper.getExecutionDetails(ctx);

    expect(details["Edited At"]).toBe(new Date("2026-03-27T18:05:47.482Z").toLocaleString());
  });

  it("omits the edit time when Linear did not set one", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ ...commentPayload, editedAt: undefined })] } },
    });
    const details = updateIssueCommentMapper.getExecutionDetails(ctx);

    expect(details["Edited At"]).toBeUndefined();
    expect(details["Comment URL"]).toBeDefined();
  });

  it("shows at most six details", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(commentPayload)] } } });
    const details = updateIssueCommentMapper.getExecutionDetails(ctx);

    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });

  it("omits the author when the comment was authored by a bot", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ ...commentPayload, user: undefined })] } },
    });
    const details = updateIssueCommentMapper.getExecutionDetails(ctx);

    expect(details["Author"]).toBeUndefined();
    expect(details["Comment URL"]).toBeDefined();
  });
});

describe("updateIssueCommentMapper.props", () => {
  it("renders the configured issue", () => {
    const props = updateIssueCommentMapper.props(
      buildComponentContext({ node: { configuration: { issue: "ENG-142", comment: "c1", body: "Updated" } } }),
    );

    expect(props.metadata).toEqual([{ icon: "hash", label: "ENG-142" }]);
  });

  it("renders no metadata when only the comment is configured", () => {
    const props = updateIssueCommentMapper.props(
      buildComponentContext({ node: { configuration: { comment: "c1", body: "Updated" } } }),
    );

    expect(props.metadata).toEqual([]);
  });

  it("does not throw when configuration is undefined", () => {
    expect(() =>
      updateIssueCommentMapper.props(buildComponentContext({ node: { configuration: undefined } })),
    ).not.toThrow();
  });
});

describe("updateIssueCommentMapper.subtitle", () => {
  it("returns the issue label from the comment payload", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: { default: [buildOutput(commentPayload)] } } });
    expect(updateIssueCommentMapper.subtitle(ctx)).toBe("ENG-142 · Deploy pipeline fails on retry");
  });

  it("falls back to a time-ago element when there is no payload", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: undefined } });
    expect(updateIssueCommentMapper.subtitle(ctx)).not.toBe("");
  });
});

describe("eventStateRegistry.updateIssueComment", () => {
  it("maps a finished success to updated", () => {
    expect(eventStateRegistry.updateIssueComment.getState(buildExecution())).toBe("updated");
  });

  it("returns running while the execution is in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });

    expect(eventStateRegistry.updateIssueComment.getState(execution)).toBe("running");
  });
});
