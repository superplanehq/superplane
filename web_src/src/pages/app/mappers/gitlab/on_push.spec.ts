import { describe, expect, it } from "vitest";

import { onPushTriggerRenderer } from "./on_push";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext, ComponentDefinition, EventInfo } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Push",
    componentName: "gitlab.onPush",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onPush",
    label: "On Push",
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
    type: "gitlab.push",
    ...overrides,
  };
}

function buildEventData() {
  return {
    object_kind: "push",
    event_name: "push",
    ref: "refs/heads/main",
    before: "95790bf891e76fee5e1747ab589903a6a1f80f22",
    after: "da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
    checkout_sha: "da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
    user_name: "John Smith",
    user_username: "jsmith",
    total_commits_count: 2,
    commits: [
      {
        id: "b6568db1bc1dcd7f8b4d5a946b0b91f9dacd7327",
        message: "Update Catalog page",
        url: "https://gitlab.example.com/jsmith/example/-/commit/b6568db1bc1dcd7f8b4d5a946b0b91f9dacd7327",
        author: { name: "John Smith" },
      },
      {
        id: "da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
        message: "Fix catalog rendering bug",
        title: "Fix catalog rendering bug",
        url: "https://gitlab.example.com/jsmith/example/-/commit/da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
        author: { name: "John Smith" },
      },
    ],
    project: {
      id: 15,
      name: "Example",
      path_with_namespace: "jsmith/example",
      web_url: "https://gitlab.example.com/jsmith/example",
    },
  };
}

// ── getTitleAndSubtitle ─────────────────────────────────────────────

describe("onPushTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses the head commit message as the title", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(onPushTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Fix catalog rendering bug");
  });

  it("falls back to a default title when there are no commits", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: {} }) };
    expect(onPushTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Push");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: undefined }) };
    expect(() => onPushTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ──────────────────────────────────────────────

describe("onPushTriggerRenderer.getRootEventValues", () => {
  it("extracts push details from event data", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onPushTriggerRenderer.getRootEventValues(ctx);
    expect(values["Branch"]).toBe("main");
    expect(values["Commit"]).toBe("Fix catalog rendering bug");
    expect(values["Author"]).toBe("John Smith");
    expect(values["Commits"]).toBe("2");
    expect(values["Commit URL"]).toBe(
      "https://gitlab.example.com/jsmith/example/-/commit/da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
    );
  });

  it("includes received at as the first value", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onPushTriggerRenderer.getRootEventValues(ctx);
    expect(Object.keys(values)[0]).toBe("Received At");
    expect(values["Received At"]).not.toBe("-");
  });

  it("returns at most 6 values", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(Object.keys(onPushTriggerRenderer.getRootEventValues(ctx)).length).toBeLessThanOrEqual(6);
  });

  it("returns placeholders when event data is missing", () => {
    const values = onPushTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Branch"]).toBe("-");
    expect(values["Commit"]).toBe("-");
    expect(values["Author"]).toBe("-");
    expect(values["Commits"]).toBe("-");
    expect(values["Commit URL"]).toBe("-");
  });
});

// ── getTriggerProps ─────────────────────────────────────────────────

describe("onPushTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Push Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onPushTriggerRenderer.getTriggerProps(ctx).title).toBe("My Push Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On Push" }),
      lastEvent: buildEvent(),
    };
    expect(onPushTriggerRenderer.getTriggerProps(ctx).title).toBe("On Push");
  });

  it("includes project metadata and configured branch filters", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { id: 15, name: "jsmith/example" } },
        configuration: { branches: [{ type: "equals", value: "main" }] },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onPushTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata?.find((m) => String(m.label).includes("jsmith/example"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("main"))).toBeDefined();
  });

  it("omits metadata when project and branches are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onPushTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([]);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({ data: buildEventData() }),
    };
    const props = onPushTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData!.title).toBe("Fix catalog rendering bug");
    expect(props.lastEventData!.state).toBe("triggered");
  });

  it("omits lastEventData when lastEvent is missing", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: undefined,
    };
    expect(onPushTriggerRenderer.getTriggerProps(ctx).lastEventData).toBeUndefined();
  });
});
