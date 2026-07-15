import { describe, expect, it } from "vitest";
import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { formatLiveNodeConfigurationIssue, resolveLiveNodePreRunStatus } from "./liveNodePreRunStatus";

const emptyActivity = { executions: [], events: [] };

function node(overrides: Partial<ComponentsNode>): ComponentsNode {
  return {
    id: "node-1",
    name: "Node",
    type: "TYPE_ACTION",
    component: "noop",
    ...overrides,
  } as ComponentsNode;
}

describe("resolveLiveNodePreRunStatus", () => {
  it("returns setup status for placeholder nodes", () => {
    expect(resolveLiveNodePreRunStatus(node({ component: undefined, name: "New Component" }), emptyActivity)).toEqual({
      title: "Select a component from the sidebar",
      purpose: "setup",
    });
  });

  it("returns setup status with a formatted description when the node has a configuration error", () => {
    expect(
      resolveLiveNodePreRunStatus(node({ errorMessage: "field 'repository' is required" }), emptyActivity, {
        configurationFields: [{ name: "repository", label: "Repository", type: "string" }],
      }),
    ).toEqual({
      title: "Finish configuring this component",
      description: "Repository is required",
      purpose: "setup",
    });
  });

  it("returns runtime status for triggers without events", () => {
    expect(resolveLiveNodePreRunStatus(node({ type: "TYPE_TRIGGER", component: "webhook" }), emptyActivity)).toEqual({
      title: "Waiting for the first event",
      purpose: "runtime",
    });
  });

  it("returns runtime status for actions without executions", () => {
    expect(resolveLiveNodePreRunStatus(node({ type: "TYPE_ACTION" }), emptyActivity)).toEqual({
      title: "Waiting for the first run...",
      purpose: "runtime",
    });
  });

  it("returns runtime status when activity exists but no run is resolved yet", () => {
    expect(
      resolveLiveNodePreRunStatus(node({ type: "TYPE_ACTION" }), {
        executions: [{ id: "exec-1" }],
        events: [],
      }),
    ).toEqual({
      title: "Waiting for the first run...",
      purpose: "runtime",
    });
  });
});

describe("formatLiveNodeConfigurationIssue", () => {
  it("maps required field errors to field labels", () => {
    expect(
      formatLiveNodeConfigurationIssue("field 'customName' is required", [
        { name: "customName", label: "Run title", type: "string" },
      ]),
    ).toBe("Run title is required");
  });

  it("maps field validation errors to field labels", () => {
    expect(formatLiveNodeConfigurationIssue("field 'branch': must be a string")).toBe("Branch: must be a string");
  });

  it("maps integration-required errors to a user-facing message", () => {
    expect(formatLiveNodeConfigurationIssue("integration is required for github.create_issue")).toBe(
      "Connect an integration instance to continue",
    );
  });

  it("returns unknown errors unchanged", () => {
    expect(formatLiveNodeConfigurationIssue("action missingcomponent not registered")).toBe(
      "action missingcomponent not registered",
    );
  });
});
