import { describe, expect, it } from "vitest";

import { createAnnotationMapper } from "./create_annotation";
import { deleteAnnotationMapper } from "./delete_annotation";
import { listAnnotationsMapper } from "./list_annotations";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

function buildNode(componentName: string, overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Grafana Mapper",
    componentName,
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
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

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "grafana.result",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildComponentContext(componentName: string, overrides?: { node?: Partial<NodeInfo> }): ComponentBaseContext {
  const node = buildNode(componentName, overrides?.node);

  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: componentName,
      label: componentName,
      description: "",
      icon: "bolt",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
  };
}

function buildDetailsContext(
  componentName: string,
  overrides?: {
    node?: Partial<NodeInfo>;
    execution?: Partial<ExecutionInfo>;
  },
): ExecutionDetailsContext {
  const node = buildNode(componentName, overrides?.node);
  return {
    nodes: [node],
    node,
    execution: buildExecution(overrides?.execution),
  };
}

describe("createAnnotationMapper", () => {
  it("uses configuration.dashboard in metadata", () => {
    const props = createAnnotationMapper.props(
      buildComponentContext("grafana.createAnnotation", {
        node: {
          configuration: {
            dashboard: "ops-overview",
            text: "Deploy completed",
            tags: ["deploy"],
          },
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ label: "Deploy completed" }),
        expect.objectContaining({ label: "deploy" }),
        expect.objectContaining({ label: "Dashboard: ops-overview" }),
      ]),
    );
  });

  it("does not throw when outputs are undefined", () => {
    const ctx = buildDetailsContext("grafana.createAnnotation", {
      node: { configuration: { dashboard: "ops-overview", text: "Deploy completed" } },
      execution: { outputs: undefined },
    });

    expect(() => createAnnotationMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

describe("listAnnotationsMapper", () => {
  it("uses configuration.dashboard in metadata", () => {
    const props = listAnnotationsMapper.props(
      buildComponentContext("grafana.listAnnotations", {
        node: {
          configuration: {
            dashboard: "ops-overview",
            text: "error budget",
            tags: ["sev2", "prod"],
          },
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ label: "Tags: sev2, prod" }),
        expect.objectContaining({ label: "Text: error budget" }),
        expect.objectContaining({ label: "Dashboard: ops-overview" }),
      ]),
    );
  });

  it("does not throw when outputs are undefined", () => {
    const ctx = buildDetailsContext("grafana.listAnnotations", {
      node: { configuration: { dashboard: "ops-overview", text: "error budget" } },
      execution: { outputs: undefined },
    });

    expect(() => listAnnotationsMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(listAnnotationsMapper.getExecutionDetails(ctx)).toMatchObject({ Count: "0" });
  });
});

describe("deleteAnnotationMapper", () => {
  it("uses configuration.annotation in metadata", () => {
    const props = deleteAnnotationMapper.props(
      buildComponentContext("grafana.deleteAnnotation", {
        node: {
          configuration: {
            annotation: "  ann-42  ",
          },
        },
      }),
    );

    expect(props.metadata).toEqual([expect.objectContaining({ label: "Annotation: ann-42" })]);
  });

  it("reads deleted annotation details from outputs", () => {
    const ctx = buildDetailsContext("grafana.deleteAnnotation", {
      node: { configuration: { annotation: "ann-42" } },
      execution: {
        outputs: {
          default: [buildOutput({ id: 42, deleted: true })],
        },
      },
    });

    expect(deleteAnnotationMapper.getExecutionDetails(ctx)).toMatchObject({ "Annotation ID": "42" });
  });
});
