import { describe, expect, it } from "vitest";

import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client/types.gen";

import { buildDashboardTriggerParameters } from "./dashboardTriggerParameters";

function makeStartNode(configuration: unknown): ComponentsNode {
  return {
    id: "node-1",
    name: "Trigger",
    type: "TYPE_TRIGGER",
    component: "start",
    configuration: configuration as ComponentsNode["configuration"],
  } as ComponentsNode;
}

describe("buildDashboardTriggerParameters", () => {
  it("returns empty parameters when no node is provided", () => {
    expect(buildDashboardTriggerParameters(undefined, "run")).toEqual({});
  });

  it("returns empty parameters for non-run hooks", () => {
    const node = makeStartNode({ templates: [{ name: "default", payload: { a: 1 } }] });
    expect(buildDashboardTriggerParameters(node, "stop")).toEqual({});
  });

  it("uses templates even when the component field is not start", () => {
    const node: ComponentsNode = {
      id: "node-2",
      name: "Other",
      type: "TYPE_TRIGGER",
      component: "github",
      configuration: { templates: [{ name: "default" }] } as ComponentsNode["configuration"],
    } as ComponentsNode;
    expect(buildDashboardTriggerParameters(node, "run")).toEqual({ template: "default" });
  });

  it("returns empty parameters when the start node has no templates", () => {
    expect(buildDashboardTriggerParameters(makeStartNode({ templates: [] }), "run")).toEqual({});
    expect(buildDashboardTriggerParameters(makeStartNode({}), "run")).toEqual({});
    expect(buildDashboardTriggerParameters(makeStartNode(undefined), "run")).toEqual({});
  });

  it("returns the first template's name when present", () => {
    const node = makeStartNode({
      templates: [
        { name: "deploy", payload: { branch: "main", env: "prod" } },
        { name: "rollback", payload: { branch: "main" } },
      ],
    });
    expect(buildDashboardTriggerParameters(node, "run")).toEqual({
      template: "deploy",
    });
  });

  it("uses the requested template when one is provided", () => {
    const node = makeStartNode({
      templates: [
        { name: "deploy", payload: { branch: "main", env: "prod" } },
        { name: "rollback", payload: { branch: "main" } },
      ],
    });
    expect(buildDashboardTriggerParameters(node, "run", "rollback")).toEqual({
      template: "rollback",
    });
  });

  it("does not depend on template payload shape", () => {
    expect(buildDashboardTriggerParameters(makeStartNode({ templates: [{ name: "deploy" }] }), "run")).toEqual({
      template: "deploy",
    });
    expect(
      buildDashboardTriggerParameters(makeStartNode({ templates: [{ name: "deploy", payload: null }] }), "run"),
    ).toEqual({ template: "deploy" });
    expect(
      buildDashboardTriggerParameters(makeStartNode({ templates: [{ name: "deploy", payload: [1, 2] }] }), "run"),
    ).toEqual({ template: "deploy" });
  });

  it("returns empty parameters when the first template has no name", () => {
    expect(buildDashboardTriggerParameters(makeStartNode({ templates: [{ payload: { a: 1 } }] }), "run")).toEqual({});
  });
});
