import { describe, expect, it } from "bun:test";

import { launchAgentMapper } from "./launch_agent";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

const node: NodeInfo = {
  id: "node-1",
  name: "Cursor agent",
  componentName: "cursor.launchAgent",
  isCollapsed: false,
};

const metadata = {
  agent: { url: "https://cursor.com/agents/agent-1", status: "RUNNING", summary: "Metadata summary" },
  target: { prUrl: "https://github.com/example/repo/pull/1", branchName: "metadata-branch" },
};

const payloadData = {
  prUrl: "https://github.com/example/repo/pull/2",
  branchName: "payload-branch",
  status: "FINISHED",
  summary: "Payload summary",
};

const metadataDetails = {
  "Cloud Agent": metadata.agent.url,
  "Pull Request": metadata.target.prUrl,
  Branch: metadata.target.branchName,
  Status: metadata.agent.status,
  Summary: metadata.agent.summary,
};

function buildDetailsContext(overrides: Partial<ExecutionInfo> = {}): ExecutionDetailsContext {
  const timestamp = new Date().toISOString();
  return {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: timestamp,
      updatedAt: timestamp,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: undefined,
      configuration: {},
      rootEvent: undefined,
      ...overrides,
    },
  };
}

function buildOutput(data: unknown, timestamp = ""): OutputPayload {
  return { type: "cursor.launchAgent", timestamp, data };
}

describe("launchAgentMapper.getExecutionDetails", () => {
  it.each([undefined, {}, { default: [] }])("preserves metadata details with missing output %j", (outputs) => {
    const context = buildDetailsContext({ metadata, outputs });

    expect(launchAgentMapper.getExecutionDetails(context)).toEqual(metadataDetails);
  });

  it("returns an empty record without metadata or payload", () => {
    expect(launchAgentMapper.getExecutionDetails(buildDetailsContext())).toEqual({});
  });

  it.each([undefined, null])("resolves payload details with absent metadata objects %j", (absent) => {
    const context = buildDetailsContext({
      metadata: { agent: absent, target: absent },
      outputs: { default: [buildOutput(payloadData)] },
    });

    expect(launchAgentMapper.getExecutionDetails(context)).toEqual({
      "Pull Request": payloadData.prUrl,
      Branch: payloadData.branchName,
      Status: payloadData.status,
      Summary: payloadData.summary,
    });
  });

  it.each([undefined, null])("preserves metadata details with absent payload data %j", (data) => {
    const context = buildDetailsContext({ metadata, outputs: { default: [buildOutput(data)] } });

    expect(launchAgentMapper.getExecutionDetails(context)).toEqual(metadataDetails);
  });

  it("prefers metadata PR and branch values, and payload status and summary values", () => {
    const context = buildDetailsContext({ metadata, outputs: { default: [buildOutput(payloadData)] } });
    const details = launchAgentMapper.getExecutionDetails(context);

    expect(details).toEqual({
      "Cloud Agent": metadata.agent.url,
      "Pull Request": metadata.target.prUrl,
      Branch: metadata.target.branchName,
      Status: payloadData.status,
      Summary: payloadData.summary,
    });
    expect(Object.keys(details)).toEqual(["Cloud Agent", "Pull Request", "Branch", "Status", "Summary"]);
  });

  it("omits empty preferred values without selecting nonempty alternates", () => {
    const context = buildDetailsContext({
      metadata: {
        agent: { ...metadata.agent, url: "" },
        target: { prUrl: "", branchName: "" },
      },
      outputs: { default: [buildOutput({ ...payloadData, status: "", summary: "" })] },
    });

    expect(launchAgentMapper.getExecutionDetails(context)).toEqual({});
  });

  it.each([undefined, null])("selects alternate values when preferred fields are %j", (absent) => {
    const context = buildDetailsContext({
      metadata: { agent: metadata.agent, target: { prUrl: absent, branchName: absent } },
      outputs: { default: [buildOutput({ ...payloadData, status: absent, summary: absent })] },
    });

    expect(launchAgentMapper.getExecutionDetails(context)).toEqual({
      "Cloud Agent": metadata.agent.url,
      "Pull Request": payloadData.prUrl,
      Branch: payloadData.branchName,
      Status: metadata.agent.status,
      Summary: metadata.agent.summary,
    });
  });

  it("uses only the first default payload and formats its emitted timestamp locally", () => {
    const timestamp = new Date().toISOString();
    const laterTimestamp = new Date(Date.now() + 60_000).toISOString();
    const context = buildDetailsContext({
      outputs: {
        default: [
          buildOutput(payloadData, timestamp),
          buildOutput(
            { prUrl: "later-pr", branchName: "later-branch", status: "ERROR", summary: "Later summary" },
            laterTimestamp,
          ),
        ],
      },
    });

    expect(launchAgentMapper.getExecutionDetails(context)).toEqual({
      "Pull Request": payloadData.prUrl,
      Branch: payloadData.branchName,
      Status: payloadData.status,
      Summary: payloadData.summary,
      "Emitted At": new Date(timestamp).toLocaleString(),
    });
  });
});
