import { describe, expect, it } from "vitest";

import { onMergeRequestTriggerRenderer } from "./on_merge_request";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext, ComponentDefinition, EventInfo } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Merge Request",
    componentName: "gitlab.onMergeRequest",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onMergeRequest",
    label: "On Merge Request",
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
    type: "gitlab.mergeRequest",
    ...overrides,
  };
}

function buildEventData() {
  return {
    object_kind: "merge_request",
    user: { id: 1, name: "Alex Garcia", username: "agarcia" },
    project: {
      id: 1,
      name: "Example Project",
      path_with_namespace: "group/example",
      web_url: "https://gitlab.example.com/group/example",
    },
    object_attributes: {
      id: 93,
      iid: 12,
      title: "Add merge request trigger",
      state: "opened",
      action: "open",
      url: "https://gitlab.example.com/group/example/-/merge_requests/12",
    },
  };
}

// ── getTitleAndSubtitle ─────────────────────────────────────────────

describe("onMergeRequestTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses merge request iid and title", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(onMergeRequestTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("!12 - Add merge request trigger");
  });

  it("falls back to default title when merge request is missing", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: {} }) };
    expect(onMergeRequestTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("! - Merge Request");
  });

  it("includes action in subtitle", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ createdAt: "", data: buildEventData() }) };
    const { subtitle } = onMergeRequestTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(subtitle).toBe("open");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: undefined }) };
    expect(() => onMergeRequestTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ──────────────────────────────────────────────

describe("onMergeRequestTriggerRenderer.getRootEventValues", () => {
  it("extracts merge request details from event data", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onMergeRequestTriggerRenderer.getRootEventValues(ctx);
    expect(values["Title"]).toBe("Add merge request trigger");
    expect(values["URL"]).toBe("https://gitlab.example.com/group/example/-/merge_requests/12");
    expect(values["Action"]).toBe("open");
    expect(values["State"]).toBe("opened");
    expect(values["Author"]).toBe("agarcia");
  });

  it("includes received at as the first value", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onMergeRequestTriggerRenderer.getRootEventValues(ctx);
    expect(Object.keys(values)[0]).toBe("Received At");
    expect(values["Received At"]).not.toBe("-");
  });

  it("returns at most 6 values", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(Object.keys(onMergeRequestTriggerRenderer.getRootEventValues(ctx)).length).toBeLessThanOrEqual(6);
  });

  it("returns placeholders when event data is missing", () => {
    const values = onMergeRequestTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Title"]).toBe("-");
    expect(values["URL"]).toBe("-");
    expect(values["Action"]).toBe("-");
    expect(values["State"]).toBe("-");
    expect(values["Author"]).toBe("-");
  });
});

// ── getTriggerProps ─────────────────────────────────────────────────

describe("onMergeRequestTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Merge Request Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onMergeRequestTriggerRenderer.getTriggerProps(ctx).title).toBe("My Merge Request Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On Merge Request" }),
      lastEvent: buildEvent(),
    };
    expect(onMergeRequestTriggerRenderer.getTriggerProps(ctx).title).toBe("On Merge Request");
  });

  it("includes project metadata and configured actions", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { id: 1, name: "group/example" } },
        configuration: { actions: ["open", "merge"] },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onMergeRequestTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata?.find((m) => String(m.label).includes("group/example"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("open, merge"))).toBeDefined();
  });

  it("omits metadata when project and actions are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onMergeRequestTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([]);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({ data: buildEventData() }),
    };
    const props = onMergeRequestTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData!.title).toBe("!12 - Add merge request trigger");
    expect(props.lastEventData!.state).toBe("triggered");
  });

  it("omits lastEventData when lastEvent is missing", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: undefined,
    };
    expect(onMergeRequestTriggerRenderer.getTriggerProps(ctx).lastEventData).toBeUndefined();
  });
});
