import { describe, expect, it } from "vitest";
import type { ComponentBaseContext, NodeInfo } from "../types";
import { runWorkflowMapper } from "./run_workflow";

function makeNode(configuration: unknown): NodeInfo {
  return {
    id: "node-1",
    name: "Run Workflow",
    componentName: "github.runWorkflow",
    isCollapsed: false,
    configuration,
    metadata: {},
  };
}

function makeContext(configuration: unknown): ComponentBaseContext {
  return {
    nodes: [],
    node: makeNode(configuration),
    componentDefinition: {
      name: "runWorkflow",
      label: "Run Workflow",
      description: "",
      icon: "github",
      color: "purple",
    },
    lastExecutions: [],
    currentUser: {
      id: "123",
      name: "John Doe",
      email: "john.doe@example.com",
      roles: ["admin"],
      groups: ["developers"],
    },
    actions: {
      invokeNodeExecutionAction: async () => {},
    },
  };
}

describe("github run_workflow mapper", () => {
  it("filters malformed inputs so specs never emit undefined badge labels", () => {
    const context = makeContext({
      inputs: [
        { name: "valid_name", value: "valid_value" },
        { name: "missing_value" },
        { value: "missing_name" },
        null,
        42,
      ],
    });

    const props = runWorkflowMapper.props(context);
    const values = props.specs?.[0]?.values;

    expect(values).toHaveLength(1);
    expect(values?.[0]?.badges?.[0]?.label).toBe("valid_name");
    expect(values?.[0]?.badges?.[1]?.label).toBe("valid_value");
  });
});
