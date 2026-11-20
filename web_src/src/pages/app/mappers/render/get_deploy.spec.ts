import { describe, expect, it } from "bun:test";
import { getDeployMapper } from "./get_deploy";
import type { ExecutionDetailsContext, ExecutionInfo } from "../types";

const timestamp = new Date().toISOString();
const localTimestamp = new Date(timestamp).toLocaleString();
const fallbackEntries = [
  ["Retrieved At", "-"],
  ["Service ID", "-"],
  ["Deploy ID", "-"],
  ["Status", "-"],
  ["Trigger", "-"],
  ["Created At", "-"],
  ["Started At", "-"],
  ["Finished At", "-"],
];
const deployment = {
  serviceId: "srv-123",
  deployId: "dep-456",
  status: "live",
  trigger: "api",
  createdAt: timestamp,
  startedAt: timestamp,
  finishedAt: timestamp,
};
const deploymentEntries = [
  ["Retrieved At", localTimestamp],
  ["Service ID", "srv-123"],
  ["Deploy ID", "dep-456"],
  ["Status", "live"],
  ["Trigger", "api"],
  ["Created At", localTimestamp],
  ["Started At", localTimestamp],
  ["Finished At", localTimestamp],
];

function output(result: Record<string, unknown>): ExecutionInfo["outputs"] {
  return { default: [{ data: result, type: "render.deploy", timestamp }] };
}

function makeDetailsContext(outputs: ExecutionInfo["outputs"], createdAt: string): ExecutionDetailsContext {
  return {
    nodes: [],
    node: { id: "n1", name: "Get Deploy", componentName: "render.getDeploy", isCollapsed: false },
    execution: {
      id: "exec-1",
      createdAt,
      updatedAt: createdAt,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
      outputs,
    },
  };
}

const cases: {
  name: string;
  outputs: ExecutionInfo["outputs"];
  createdAt: string;
  entries: string[][];
}[] = [
  { name: "missing outputs", outputs: undefined, createdAt: "", entries: fallbackEntries },
  { name: "missing default output", outputs: {}, createdAt: "", entries: fallbackEntries },
  { name: "empty default output array", outputs: { default: [] }, createdAt: "", entries: fallbackEntries },
  { name: "missing first result", outputs: { default: [{}] }, createdAt: "", entries: fallbackEntries },
  {
    name: "commit deployment",
    outputs: output({ ...deployment, commit: { id: "abc123", message: "Deploy backend" } }),
    createdAt: timestamp,
    entries: [...deploymentEntries, ["Commit ID", "abc123"], ["Commit Message", "Deploy backend"]],
  },
  {
    name: "image deployment",
    outputs: output({ ...deployment, image: { ref: "registry/backend:latest", sha: "sha256:123" } }),
    createdAt: timestamp,
    entries: [...deploymentEntries, ["Image Ref", "registry/backend:latest"], ["Image SHA", "sha256:123"]],
  },
  {
    name: "rollback deployment",
    outputs: output({ ...deployment, rollbackToDeployId: "dep-previous" }),
    createdAt: timestamp,
    entries: [...deploymentEntries, ["Rollback To", "dep-previous"]],
  },
  {
    name: "all optional groups in display order",
    outputs: output({
      ...deployment,
      updatedAt: timestamp,
      image: { sha: "sha256:123", ref: "registry/backend:latest" },
      commit: { message: "Deploy backend", id: "abc123" },
      rollbackToDeployId: "dep-previous",
    }),
    createdAt: timestamp,
    entries: [
      ...deploymentEntries,
      ["Rollback To", "dep-previous"],
      ["Commit ID", "abc123"],
      ["Commit Message", "Deploy backend"],
      ["Image Ref", "registry/backend:latest"],
      ["Image SHA", "sha256:123"],
      ["Updated At", localTimestamp],
    ],
  },
  {
    name: "empty nested objects and absent timestamps",
    outputs: output({ commit: {}, image: {} }),
    createdAt: "",
    entries: fallbackEntries,
  },
  {
    name: "empty required and optional values",
    outputs: output({
      serviceId: "",
      deployId: "",
      status: "",
      trigger: "",
      createdAt: "",
      startedAt: "",
      finishedAt: "",
      updatedAt: "",
      rollbackToDeployId: "",
      commit: { id: "", message: "" },
      image: { ref: "", sha: "" },
    }),
    createdAt: "",
    entries: fallbackEntries,
  },
  {
    name: "invalid deployment timestamps",
    outputs: output({ createdAt: "invalid", startedAt: "invalid", finishedAt: "invalid", updatedAt: "invalid" }),
    createdAt: "",
    entries: [...fallbackEntries, ["Updated At", "-"]],
  },
  {
    name: "partial nested objects omit only empty fields",
    outputs: output({
      commit: { id: "", message: "Deploy backend" },
      image: { ref: "registry/backend:latest", sha: "" },
    }),
    createdAt: "",
    entries: [...fallbackEntries, ["Commit Message", "Deploy backend"], ["Image Ref", "registry/backend:latest"]],
  },
  {
    name: "invalid retrieval timestamp retains its separate formatting",
    outputs: undefined,
    createdAt: "invalid",
    entries: [["Retrieved At", new Date("invalid").toLocaleString()], ...fallbackEntries.slice(1)],
  },
];

describe("getDeployMapper.getExecutionDetails", () => {
  it.each(cases)("preserves ordered details for $name", ({ outputs, createdAt, entries }) => {
    const details = getDeployMapper.getExecutionDetails(makeDetailsContext(outputs, createdAt));
    expect(Object.entries(details)).toEqual(entries);
  });
});
