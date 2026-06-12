import { describe, expect, it } from "vitest";

import { sshMapper, SSH_STATE_REGISTRY } from "./ssh";
import type {
  ComponentBaseContext,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "./types";

function buildNode(configuration: Record<string, unknown> = {}): NodeInfo {
  return {
    id: "node-1",
    name: "SSH",
    componentName: "ssh",
    isCollapsed: true,
    configuration,
    metadata: {},
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "ssh.command.executed",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution({
  metadata = {},
  outputs,
  state = "STATE_FINISHED",
  result = "RESULT_PASSED",
  resultReason = "RESULT_REASON_OK",
  resultMessage = "",
  updatedAt = new Date().toISOString(),
}: {
  metadata?: Record<string, unknown>;
  outputs?: { success?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] };
  state?: ExecutionInfo["state"];
  result?: ExecutionInfo["result"];
  resultReason?: ExecutionInfo["resultReason"];
  resultMessage?: string;
  updatedAt?: string;
}): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt,
    state,
    result,
    resultReason,
    resultMessage,
    metadata,
    configuration: {},
    rootEvent: undefined,
    outputs,
  };
}

describe("SSH_STATE_REGISTRY", () => {
  it("uses failed output payload exit codes", () => {
    const execution = buildExecution({
      outputs: {
        failed: [buildOutput({ exitCode: 23, stderr: "boom" })],
      },
    });

    expect(SSH_STATE_REGISTRY.getState(execution)).toBe("failed");
  });

  it("accepts string exit codes from metadata", () => {
    const execution = buildExecution({
      metadata: {
        result: {
          exitCode: "0",
        },
      },
    });

    expect(SSH_STATE_REGISTRY.getState(execution)).toBe("success");
  });

  it("falls back to success when no exit code can be inferred", () => {
    const execution = buildExecution({
      metadata: {},
      outputs: {},
    });

    expect(SSH_STATE_REGISTRY.getState(execution)).toBe("success");
  });
});

function buildComponentCtx(node: NodeInfo): ComponentBaseContext {
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "ssh",
      label: "SSH Command",
      description: "",
      icon: "terminal",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("sshMapper metadata preview", () => {
  it("shows joined commands for inline mode", () => {
    const node = buildNode({
      host: "example.com",
      username: "root",
      commandSource: "inline",
      commands: "echo hi\nls -la",
    });

    const props = sshMapper.props(buildComponentCtx(node));

    expect(props.metadata).toContainEqual({ icon: "terminal", label: "echo hi && ls -la" });
  });

  it("shows the file path for file mode", () => {
    const node = buildNode({
      host: "example.com",
      username: "root",
      commandSource: "file",
      commandFile: "scripts/deploy.sh",
      commands: "stale inline value that should be hidden",
    });

    const props = sshMapper.props(buildComponentCtx(node));

    expect(props.metadata).toContainEqual({ icon: "file-code", label: "scripts/deploy.sh" });
    expect(props.metadata).not.toContainEqual({ icon: "terminal", label: "stale inline value that should be hidden" });
  });

  it("uses exact matching so a padded commandSource is not treated as file mode (matches UI conditions + backend)", () => {
    const node = buildNode({
      host: "example.com",
      username: "root",
      commandSource: "\tfile\n",
      commandFile: "scripts/deploy.sh",
      commands: "echo inline fallback",
    });

    const props = sshMapper.props(buildComponentCtx(node));

    expect(props.metadata).not.toContainEqual({ icon: "file-code", label: "scripts/deploy.sh" });
    expect(props.metadata).toContainEqual({ icon: "terminal", label: "echo inline fallback" });
  });

  it("hides the stale inline preview in file mode when commandFile is empty", () => {
    const node = buildNode({
      host: "example.com",
      username: "root",
      commandSource: "file",
      commandFile: "",
      commands: "stale inline value that should be hidden",
    });

    const props = sshMapper.props(buildComponentCtx(node));

    expect(props.metadata).not.toContainEqual({ icon: "terminal", label: "stale inline value that should be hidden" });
    expect(props.metadata).not.toContainEqual({ icon: "file-code", label: "" });
  });

  it("hides the stale inline preview in file mode when commandFile is missing", () => {
    const node = buildNode({
      host: "example.com",
      username: "root",
      commandSource: "file",
      commands: "stale inline value that should be hidden",
    });

    const props = sshMapper.props(buildComponentCtx(node));

    expect(props.metadata).not.toContainEqual({ icon: "terminal", label: "stale inline value that should be hidden" });
  });

  it("falls back to inline preview when commandSource is missing (legacy nodes)", () => {
    const node = buildNode({
      host: "example.com",
      username: "root",
      commands: "echo legacy",
    });

    const props = sshMapper.props(buildComponentCtx(node));

    expect(props.metadata).toContainEqual({ icon: "terminal", label: "echo legacy" });
  });
});

describe("sshMapper", () => {
  it("reads execution details from output payloads when metadata result is missing", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        metadata: {
          host: "example.com",
          user: "root",
        },
        outputs: {
          failed: [buildOutput({ exitCode: 7, stdout: "ok", stderr: "boom" })],
        },
      }),
    };

    expect(sshMapper.getExecutionDetails(ctx)).toMatchObject({
      Host: "root@example.com",
      "Exit code": "7",
      Stdout: "ok",
      Stderr: "boom",
    });
  });

  it("renders exit code in subtitle from the failed output payload", () => {
    const node = buildNode();
    const ctx: SubtitleContext = {
      node,
      execution: buildExecution({
        outputs: {
          failed: [buildOutput({ exitCode: 2 })],
        },
      }),
    };

    const subtitle = sshMapper.subtitle(ctx);
    expect(typeof subtitle).not.toBe("string");
  });
});
