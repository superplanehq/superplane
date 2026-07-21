import { describe, expect, it } from "vitest";

import { onMRDiffNoteTriggerRenderer } from "./on_mr_diff_note";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext, ComponentDefinition, EventInfo } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On MR Diff Note",
    componentName: "gitlab.onMRDiffNote",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onMRDiffNote",
    label: "On MR Diff Note",
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
    type: "gitlab.mrDiffNote",
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
      id: 1401,
      note: "This variable name is misleading, can we rename it?",
      noteable_type: "MergeRequest",
      type: "DiffNote",
      url: "https://gitlab.example.com/group/example/-/merge_requests/12#note_1401",
      position: {
        new_path: "src/handlers/login.go",
        old_path: "src/handlers/login.go",
        new_line: 10,
        old_line: 10,
      },
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

describe("onMRDiffNoteTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses merge request iid and title", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(onMRDiffNoteTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("!12 - Add merge request trigger");
  });

  it("falls back to default title when merge request is missing", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: {} }) };
    expect(onMRDiffNoteTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("! - MR Diff Note");
  });

  it("includes comment author in subtitle", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ createdAt: "", data: buildEventData() }) };
    const { subtitle } = onMRDiffNoteTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(subtitle).toBe("By agarcia");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: undefined }) };
    expect(() => onMRDiffNoteTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ──────────────────────────────────────────────

describe("onMRDiffNoteTriggerRenderer.getRootEventValues", () => {
  it("extracts comment and diff position details from event data", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onMRDiffNoteTriggerRenderer.getRootEventValues(ctx);
    expect(values["Comment"]).toBe("This variable name is misleading, can we rename it?");
    expect(values["Diff Location"]).toBe("src/handlers/login.go:10");
    expect(values["Author"]).toBe("agarcia");
    expect(values["Merge Request"]).toBe("!12 - Add merge request trigger");
    expect(values["Project"]).toBe("group/example");
  });

  it("includes received at as the first value", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onMRDiffNoteTriggerRenderer.getRootEventValues(ctx);
    expect(Object.keys(values)[0]).toBe("Received At");
    expect(values["Received At"]).not.toBe("-");
  });

  it("returns at most 6 values", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(Object.keys(onMRDiffNoteTriggerRenderer.getRootEventValues(ctx)).length).toBeLessThanOrEqual(6);
  });

  it("returns placeholders when event data is missing", () => {
    const values = onMRDiffNoteTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Comment"]).toBe("-");
    expect(values["Diff Location"]).toBe("-");
    expect(values["Author"]).toBe("-");
    expect(values["Merge Request"]).toBe("-");
    expect(values["Project"]).toBe("-");
  });
});

// ── getTriggerProps ─────────────────────────────────────────────────

describe("onMRDiffNoteTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Diff Note Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onMRDiffNoteTriggerRenderer.getTriggerProps(ctx).title).toBe("My Diff Note Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On MR Diff Note" }),
      lastEvent: buildEvent(),
    };
    expect(onMRDiffNoteTriggerRenderer.getTriggerProps(ctx).title).toBe("On MR Diff Note");
  });

  it("includes project metadata and content filter", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { id: 1, name: "group/example" } },
        configuration: { contentFilter: "/fix" },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onMRDiffNoteTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata?.find((m) => String(m.label).includes("group/example"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("/fix"))).toBeDefined();
  });

  it("omits metadata when project and content filter are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onMRDiffNoteTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([]);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({ data: buildEventData() }),
    };
    const props = onMRDiffNoteTriggerRenderer.getTriggerProps(ctx);
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
    expect(onMRDiffNoteTriggerRenderer.getTriggerProps(ctx).lastEventData).toBeUndefined();
  });
});
