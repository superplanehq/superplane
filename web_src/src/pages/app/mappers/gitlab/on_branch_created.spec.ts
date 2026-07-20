import { describe, expect, it } from "vitest";

import { onBranchCreatedTriggerRenderer } from "./on_branch_created";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext, ComponentDefinition, EventInfo } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Branch Created",
    componentName: "gitlab.onBranchCreated",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onBranchCreated",
    label: "On Branch Created",
    description: "",
    icon: "gitlab",
    color: "orange",
    ...overrides,
  };
}

function buildEvent(overrides?: Partial<NonNullable<EventInfo>>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date().toISOString(),
    data: {},
    nodeId: "node-1",
    type: "gitlab.branchCreated",
    ...overrides,
  };
}

function buildEventData() {
  return {
    object_kind: "push",
    event_name: "push",
    ref: "refs/heads/feature/new-feature",
    before: "0000000000000000000000000000000000000000",
    after: "da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
    user_name: "John Smith",
    user_username: "jsmith",
    project: {
      id: 15,
      name: "Example",
      path_with_namespace: "jsmith/example",
      web_url: "https://gitlab.example.com/jsmith/example",
    },
  };
}

// ── getTitleAndSubtitle ─────────────────────────────────────────────

describe("onBranchCreatedTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses the branch name in the title", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(onBranchCreatedTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Branch: feature/new-feature");
  });

  it("falls back to a default title when the ref is missing", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: {} }) };
    expect(onBranchCreatedTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Branch Created");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: undefined }) };
    expect(() => onBranchCreatedTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ──────────────────────────────────────────────

describe("onBranchCreatedTriggerRenderer.getRootEventValues", () => {
  it("extracts branch details from event data", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onBranchCreatedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Branch"]).toBe("feature/new-feature");
    expect(values["Project"]).toBe("jsmith/example");
    expect(values["Author"]).toBe("John Smith");
    expect(values["SHA"]).toBe("da156088");
    expect(values["Branch URL"]).toBe("https://gitlab.example.com/jsmith/example/-/tree/feature/new-feature");
  });

  it("URL-encodes branch names with URL-significant characters", () => {
    const data = { ...buildEventData(), ref: "refs/heads/fix#42" };
    const ctx: TriggerEventContext = { event: buildEvent({ data }) };
    const values = onBranchCreatedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Branch"]).toBe("fix#42");
    expect(values["Branch URL"]).toBe("https://gitlab.example.com/jsmith/example/-/tree/fix%2342");
  });

  it("includes received at as the first value", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onBranchCreatedTriggerRenderer.getRootEventValues(ctx);
    expect(Object.keys(values)[0]).toBe("Received At");
    expect(values["Received At"]).not.toBe("-");
  });

  it("returns at most 6 values", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(Object.keys(onBranchCreatedTriggerRenderer.getRootEventValues(ctx)).length).toBeLessThanOrEqual(6);
  });

  it("returns placeholders when event data is missing", () => {
    const values = onBranchCreatedTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Branch"]).toBe("-");
    expect(values["Project"]).toBe("-");
    expect(values["Author"]).toBe("-");
    expect(values["SHA"]).toBe("-");
    expect(values["Branch URL"]).toBe("-");
  });
});

// ── getTriggerProps ─────────────────────────────────────────────────

describe("onBranchCreatedTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Branch Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onBranchCreatedTriggerRenderer.getTriggerProps(ctx).title).toBe("My Branch Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On Branch Created" }),
      lastEvent: buildEvent(),
    };
    expect(onBranchCreatedTriggerRenderer.getTriggerProps(ctx).title).toBe("On Branch Created");
  });

  it("includes project metadata and configured branch filters", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { id: 15, name: "jsmith/example" } },
        configuration: { branches: [{ type: "matches", value: "feature/.*" }] },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onBranchCreatedTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata?.find((m) => String(m.label).includes("jsmith/example"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("feature/.*"))).toBeDefined();
  });

  it("omits metadata when project and branches are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onBranchCreatedTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([]);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({ data: buildEventData() }),
    };
    const props = onBranchCreatedTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData!.title).toBe("Branch: feature/new-feature");
    expect(props.lastEventData!.state).toBe("triggered");
  });

  it("omits lastEventData when lastEvent is missing", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: undefined,
    };
    expect(onBranchCreatedTriggerRenderer.getTriggerProps(ctx).lastEventData).toBeUndefined();
  });
});
