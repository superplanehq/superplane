import { expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import { deployReleaseMapper } from "./deploy_release";

const now = new Date();
const created = new Date(now.getTime() - 120_000);
const completed = new Date(now.getTime() - 60_000);
const names = { projectName: "Website", environmentName: "Production", releaseName: "1.2.3" };
const success = {
  deploymentId: "Deployments-1",
  taskState: "Success",
  projectId: "Projects-1",
  environmentId: "Environments-1",
  releaseId: "Releases-1",
  created: created.toISOString(),
  completedTime: completed.toISOString(),
  duration: "00:01:00",
};
const failed = {
  deploymentId: "Deployments-2",
  taskState: "Failed",
  projectId: "Projects-2",
  environmentId: "Environments-2",
  releaseId: "Releases-2",
  created: completed.toISOString(),
  completedTime: now.toISOString(),
  duration: "00:02:00",
};

function output(data: unknown): OutputPayload {
  return { type: "octopus.deployment", timestamp: now.toISOString(), data };
}

const emptyDetails = ["-", "-", "-", "-", "-", now.toLocaleString(), "-", "-"];
const failedDetails = [
  "Deployments-2",
  "Failed",
  "Projects-2",
  "Environments-2",
  "Releases-2",
  completed.toLocaleString(),
  now.toLocaleString(),
  "00:02:00",
];

interface DetailsCase {
  name: string;
  outputs?: ExecutionInfo["outputs"];
  metadata?: NodeInfo["metadata"];
  createdAt?: string;
  expected: string[];
}

const cases: DetailsCase[] = [
  {
    name: "success precedes conflicting failed output and metadata names precede IDs",
    outputs: { success: [output(success)], failed: [output(failed)] },
    metadata: names,
    expected: [
      "Deployments-1",
      "Success",
      "Website",
      "Production",
      "1.2.3",
      created.toLocaleString(),
      completed.toLocaleString(),
      "00:01:00",
    ],
  },
  {
    name: "failed-only output uses IDs when metadata is missing",
    outputs: { failed: [output(failed)] },
    expected: failedDetails,
  },
  {
    name: "empty metadata names fall back to failed output IDs",
    outputs: { failed: [output(failed)] },
    metadata: { projectName: "", environmentName: "", releaseName: "" },
    expected: failedDetails,
  },
  { name: "missing outputs and metadata retain dash and execution-time fallbacks", expected: emptyDetails },
  {
    name: "empty output arrays and null metadata retain fallbacks",
    outputs: { success: [], failed: [] },
    metadata: null,
    expected: emptyDetails,
  },
  {
    name: "absent payload data retains fallbacks",
    outputs: { success: [output(undefined)], failed: [output(undefined)] },
    metadata: {},
    expected: emptyDetails,
  },
  {
    name: "metadata names display without output",
    metadata: names,
    expected: ["-", "-", "Website", "Production", "1.2.3", now.toLocaleString(), "-", "-"],
  },
  {
    name: "an empty success object still precedes failed data",
    outputs: { success: [output({})], failed: [output(failed)] },
    expected: emptyDetails,
  },
  {
    name: "null success data falls back to failed data",
    outputs: { success: [output(null)], failed: [output(failed)] },
    metadata: null,
    expected: failedDetails,
  },
  {
    name: "undefined success data falls back to failed data",
    outputs: { success: [output(undefined)], failed: [output(failed)] },
    expected: failedDetails,
  },
  {
    name: "empty display values retain dash and execution-time fallbacks",
    outputs: {
      success: [
        output({
          deploymentId: "",
          taskState: "",
          projectId: "",
          environmentId: "",
          releaseId: "",
          created: "",
          completedTime: "",
          duration: "",
        }),
      ],
    },
    expected: emptyDetails,
  },
  {
    name: "invalid output timestamps retain dashes instead of execution-time fallbacks",
    outputs: { success: [output({ ...success, created: "invalid", completedTime: "invalid" })] },
    expected: ["Deployments-1", "Success", "Projects-1", "Environments-1", "Releases-1", "-", "-", "00:01:00"],
  },
  {
    name: "missing execution time leaves Created and Completed At as dashes",
    createdAt: "",
    expected: ["-", "-", "-", "-", "-", "-", "-", "-"],
  },
];

it.each(cases)("deployReleaseMapper.getExecutionDetails: $name", ({ outputs, metadata, createdAt, expected }) => {
  const node: NodeInfo = {
    id: "node-1",
    name: "Deploy Release",
    componentName: "octopus.deployRelease",
    isCollapsed: false,
    metadata,
  };
  const context: ExecutionDetailsContext = {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: createdAt ?? now.toISOString(),
      updatedAt: now.toISOString(),
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

  const details = deployReleaseMapper.getExecutionDetails(context);
  expect(Object.keys(details)).toEqual([
    "Deployment ID",
    "Task State",
    "Project",
    "Environment",
    "Release",
    "Created",
    "Completed At",
    "Duration",
  ]);
  expect(Object.values(details)).toEqual(expected);
});
