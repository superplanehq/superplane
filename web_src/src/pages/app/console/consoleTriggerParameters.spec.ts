import { describe, expect, it } from "vitest";

import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client/types.gen";

import { buildConsoleTriggerParameters, resolveStartTemplate } from "./consoleTriggerParameters";

function makeStartNode(configuration: unknown): ComponentsNode {
  return {
    id: "node-1",
    name: "Trigger",
    type: "TYPE_TRIGGER",
    component: "start",
    configuration: configuration as ComponentsNode["configuration"],
  } as ComponentsNode;
}

describe("buildConsoleTriggerParameters", () => {
  it("returns empty parameters when no node is provided", () => {
    expect(buildConsoleTriggerParameters(undefined, "run")).toEqual({});
  });

  it("returns empty parameters for non-run hooks", () => {
    const node = makeStartNode({ templates: [{ name: "default", payload: { a: 1 } }] });
    expect(buildConsoleTriggerParameters(node, "stop")).toEqual({});
  });

  it("uses templates even when the component field is not start", () => {
    const node: ComponentsNode = {
      id: "node-2",
      name: "Other",
      type: "TYPE_TRIGGER",
      component: "github",
      configuration: { templates: [{ name: "default" }] } as ComponentsNode["configuration"],
    } as ComponentsNode;
    expect(buildConsoleTriggerParameters(node, "run")).toEqual({ template: "default" });
  });

  it("returns empty parameters when the start node has no templates", () => {
    expect(buildConsoleTriggerParameters(makeStartNode({ templates: [] }), "run")).toEqual({});
    expect(buildConsoleTriggerParameters(makeStartNode({}), "run")).toEqual({});
    expect(buildConsoleTriggerParameters(makeStartNode(undefined), "run")).toEqual({});
  });

  it("returns the first template's name when present", () => {
    const node = makeStartNode({
      templates: [
        { name: "deploy", payload: { branch: "main", env: "prod" } },
        { name: "rollback", payload: { branch: "main" } },
      ],
    });
    expect(buildConsoleTriggerParameters(node, "run")).toEqual({
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
    expect(buildConsoleTriggerParameters(node, "run", "rollback")).toEqual({
      template: "rollback",
    });
  });

  it("does not depend on template payload shape", () => {
    expect(buildConsoleTriggerParameters(makeStartNode({ templates: [{ name: "deploy" }] }), "run")).toEqual({
      template: "deploy",
    });
    expect(
      buildConsoleTriggerParameters(makeStartNode({ templates: [{ name: "deploy", payload: null }] }), "run"),
    ).toEqual({ template: "deploy" });
    expect(
      buildConsoleTriggerParameters(makeStartNode({ templates: [{ name: "deploy", payload: [1, 2] }] }), "run"),
    ).toEqual({ template: "deploy" });
  });

  it("returns empty parameters when the first template has no name", () => {
    expect(buildConsoleTriggerParameters(makeStartNode({ templates: [{ payload: { a: 1 } }] }), "run")).toEqual({});
  });
});

describe("resolveStartTemplate", () => {
  it("returns undefined when the node is missing or has no templates", () => {
    expect(resolveStartTemplate(undefined)).toBeUndefined();
    expect(resolveStartTemplate(makeStartNode(undefined))).toBeUndefined();
    expect(resolveStartTemplate(makeStartNode({ templates: [] }))).toBeUndefined();
  });

  it("returns the requested template by name when present", () => {
    const node = makeStartNode({
      templates: [
        { name: "deploy", payload: { a: 1 } },
        { name: "rollback", payload: { b: 2 } },
      ],
    });
    expect(resolveStartTemplate(node, "rollback")).toEqual({ name: "rollback", payload: { b: 2 } });
  });

  it("falls back to the first named template when no name is provided or the match is missing", () => {
    const node = makeStartNode({
      templates: [
        { name: "deploy", payload: { a: 1 } },
        { name: "rollback", payload: { b: 2 } },
      ],
    });
    expect(resolveStartTemplate(node)).toEqual({ name: "deploy", payload: { a: 1 } });
    expect(resolveStartTemplate(node, "unknown")).toEqual({ name: "deploy", payload: { a: 1 } });
  });

  it("exposes parameter declarations so the dialog can render the form", () => {
    const node = makeStartNode({
      templates: [
        {
          name: "manual",
          payload: { reason: "console" },
          parameters: [{ name: "branch", type: "string", defaultString: "main" }],
        },
      ],
    });
    expect(resolveStartTemplate(node)?.parameters).toEqual([{ name: "branch", type: "string", defaultString: "main" }]);
  });
});
