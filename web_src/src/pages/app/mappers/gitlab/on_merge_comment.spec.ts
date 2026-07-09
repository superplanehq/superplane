import { describe, expect, it } from "vitest";

import { onMergeCommentTriggerRenderer } from "./on_merge_comment";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext, ComponentDefinition, EventInfo } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Merge Comment",
    componentName: "gitlab.onMergeComment",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onMergeComment",
    label: "On Merge Comment",
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
    type: "gitlab.mergeComment",
    ...overrides,
  };
}

function buildEventData() {
  return {
    object_kind: "note",
    user: { id: 1, name: "Alex Garcia", username: "agarcia" },
    project: {
      id: 1,
      name: "Example Project",
      path_with_namespace: "group/example",
      web_url: "https://gitlab.example.com/group/example",
    },
    object_attributes: {
      id: 1244,
      note: "/deploy to staging",
      noteable_type: "MergeRequest",
      url: "https://gitlab.example.com/group/example/-/merge_requests/12#note_1244",
    },
    merge_request: {
      id: 93,
      iid: 12,
      title: "Add merge request trigger",
      state: "opened",
      url: "https://gitlab.example.com/group/example/-/merge_requests/12",
    },
  };
}

// ── getTitleAndSubtitle ─────────────────────────────────────────────

describe("onMergeCommentTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses merge request iid and title", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(onMergeCommentTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("!12 - Add merge request trigger");
  });

  it("falls back to default title when merge request is missing", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: {} }) };
    expect(onMergeCommentTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("! - Merge Comment");
  });

  it("includes comment author in subtitle", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ createdAt: "", data: buildEventData() }) };
    const { subtitle } = onMergeCommentTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(subtitle).toBe("By agarcia");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: undefined }) };
    expect(() => onMergeCommentTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ──────────────────────────────────────────────

describe("onMergeCommentTriggerRenderer.getRootEventValues", () => {
  it("extracts comment details from event data", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onMergeCommentTriggerRenderer.getRootEventValues(ctx);
    expect(values["Comment"]).toBe("/deploy to staging");
    expect(values["Comment URL"]).toBe("https://gitlab.example.com/group/example/-/merge_requests/12#note_1244");
    expect(values["Author"]).toBe("agarcia");
    expect(values["Merge Request"]).toBe("!12 - Add merge request trigger");
    expect(values["Project"]).toBe("group/example");
  });

  it("includes received at as the first value", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onMergeCommentTriggerRenderer.getRootEventValues(ctx);
    expect(Object.keys(values)[0]).toBe("Received At");
    expect(values["Received At"]).not.toBe("-");
  });

  it("returns at most 6 values", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(Object.keys(onMergeCommentTriggerRenderer.getRootEventValues(ctx)).length).toBeLessThanOrEqual(6);
  });

  it("returns placeholders when event data is missing", () => {
    const values = onMergeCommentTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Comment"]).toBe("-");
    expect(values["Comment URL"]).toBe("-");
    expect(values["Author"]).toBe("-");
    expect(values["Merge Request"]).toBe("-");
    expect(values["Project"]).toBe("-");
  });
});

// ── getTriggerProps ─────────────────────────────────────────────────

describe("onMergeCommentTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Merge Comment Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onMergeCommentTriggerRenderer.getTriggerProps(ctx).title).toBe("My Merge Comment Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On Merge Comment" }),
      lastEvent: buildEvent(),
    };
    expect(onMergeCommentTriggerRenderer.getTriggerProps(ctx).title).toBe("On Merge Comment");
  });

  it("includes project metadata and content filter", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { id: 1, name: "group/example" } },
        configuration: { contentFilter: "/deploy" },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onMergeCommentTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata?.find((m) => String(m.label).includes("group/example"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("/deploy"))).toBeDefined();
  });

  it("omits metadata when project and content filter are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onMergeCommentTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([]);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({ data: buildEventData() }),
    };
    const props = onMergeCommentTriggerRenderer.getTriggerProps(ctx);
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
    expect(onMergeCommentTriggerRenderer.getTriggerProps(ctx).lastEventData).toBeUndefined();
  });
});
