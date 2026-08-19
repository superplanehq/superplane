import { describe, expect, it } from "vitest";

import { addIssueCommentMapper } from "./add_issue_comment";
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
    componentName: "linear.addIssueComment",
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
    createdAt: new Date("2026-03-26T19:29:35Z").toISOString(),
    updatedAt: new Date("2026-03-26T19:29:35Z").toISOString(),
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
      name: "linear.addIssueComment",
      label: "Add Issue Comment",
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
  body: "Deploy pipeline is green again.",
  url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry#comment-d3f9b0a2",
  createdAt: "2026-03-27T16:42:11.123Z",
  updatedAt: "2026-03-27T16:42:11.123Z",
  user: { id: "u1", name: "Ada Lovelace", displayName: "ada" },
  issue: {
    id: "i1",
    identifier: "ENG-142",
    title: "Deploy pipeline fails on retry",
    url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry",
  },
};

describe("addIssueCommentMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => addIssueCommentMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("always includes Executed At first", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(commentPayload)] } } });
    const details = addIssueCommentMapper.getExecutionDetails(ctx);

    expect(details["Executed At"]).toBeDefined();
    expect(Object.keys(details)[0]).toBe("Executed At");
  });

  it("extracts the comment fields that matter, including the link", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(commentPayload)] } } });
    const details = addIssueCommentMapper.getExecutionDetails(ctx);

    expect(details["Comment URL"]).toBe(
      "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry#comment-d3f9b0a2",
    );
    expect(details["Issue"]).toBe("ENG-142 · Deploy pipeline fails on retry");
    expect(details["Author"]).toBe("ada");
    expect(details["Comment"]).toBe("Deploy pipeline is green again.");
  });

  it("shows at most six details", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(commentPayload)] } } });
    const details = addIssueCommentMapper.getExecutionDetails(ctx);

    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });

  it("omits the author when the comment was authored by a bot", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ ...commentPayload, user: undefined })] } },
    });
    const details = addIssueCommentMapper.getExecutionDetails(ctx);

    expect(details["Author"]).toBeUndefined();
    expect(details["Comment URL"]).toBeDefined();
  });
});

describe("addIssueCommentMapper.props", () => {
  it("renders the configured issue", () => {
    const props = addIssueCommentMapper.props(
      buildComponentContext({ node: { configuration: { issue: "ENG-142", body: "Looks good" } } }),
    );

    expect(props.metadata).toEqual([{ icon: "hash", label: "ENG-142" }]);
  });

  it("does not throw when configuration is undefined", () => {
    expect(() =>
      addIssueCommentMapper.props(buildComponentContext({ node: { configuration: undefined } })),
    ).not.toThrow();
  });
});

describe("addIssueCommentMapper.subtitle", () => {
  it("returns the issue label from the comment payload", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: { default: [buildOutput(commentPayload)] } } });
    expect(addIssueCommentMapper.subtitle(ctx)).toBe("ENG-142 · Deploy pipeline fails on retry");
  });

  it("falls back to a time-ago element when there is no payload", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: undefined } });
    expect(addIssueCommentMapper.subtitle(ctx)).not.toBe("");
  });
});

describe("eventStateRegistry.addIssueComment", () => {
  it("maps a finished success to commented", () => {
    expect(eventStateRegistry.addIssueComment.getState(buildExecution())).toBe("commented");
  });

  it("returns running while the execution is in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });

    expect(eventStateRegistry.addIssueComment.getState(execution)).toBe("running");
  });
});
