import { describe, expect, it } from "vitest";

import { onIssueCommentTriggerRenderer } from "./on_issue_comment";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext, ComponentDefinition, EventInfo } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Issue Comment",
    componentName: "gitlab.onIssueComment",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onIssueComment",
    label: "On Issue Comment",
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
    type: "gitlab.issueComment",
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
      id: 1355,
      note: "/sp-investigate",
      noteable_type: "Issue",
      url: "https://gitlab.example.com/group/example/-/issues/8#note_1355",
    },
    issue: {
      id: 45,
      iid: 8,
      title: "Login page throws 500 on invalid credentials",
      state: "opened",
      url: "https://gitlab.example.com/group/example/-/issues/8",
    },
  };
}

// ── getTitleAndSubtitle ─────────────────────────────────────────────

describe("onIssueCommentTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses issue iid and title", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(onIssueCommentTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe(
      "#8 - Login page throws 500 on invalid credentials",
    );
  });

  it("falls back to default title when issue is missing", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: {} }) };
    expect(onIssueCommentTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("# - Issue Comment");
  });

  it("includes comment author in subtitle", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ createdAt: "", data: buildEventData() }) };
    const { subtitle } = onIssueCommentTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(subtitle).toBe("By agarcia");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: undefined }) };
    expect(() => onIssueCommentTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ──────────────────────────────────────────────

describe("onIssueCommentTriggerRenderer.getRootEventValues", () => {
  it("extracts comment details from event data", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onIssueCommentTriggerRenderer.getRootEventValues(ctx);
    expect(values["Comment"]).toBe("/sp-investigate");
    expect(values["Comment URL"]).toBe("https://gitlab.example.com/group/example/-/issues/8#note_1355");
    expect(values["Author"]).toBe("agarcia");
    expect(values["Issue"]).toBe("#8 - Login page throws 500 on invalid credentials");
    expect(values["Project"]).toBe("group/example");
  });

  it("includes received at as the first value", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onIssueCommentTriggerRenderer.getRootEventValues(ctx);
    expect(Object.keys(values)[0]).toBe("Received At");
    expect(values["Received At"]).not.toBe("-");
  });

  it("returns at most 6 values", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(Object.keys(onIssueCommentTriggerRenderer.getRootEventValues(ctx)).length).toBeLessThanOrEqual(6);
  });

  it("returns placeholders when event data is missing", () => {
    const values = onIssueCommentTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Comment"]).toBe("-");
    expect(values["Comment URL"]).toBe("-");
    expect(values["Author"]).toBe("-");
    expect(values["Issue"]).toBe("-");
    expect(values["Project"]).toBe("-");
  });
});

// ── getTriggerProps ─────────────────────────────────────────────────

describe("onIssueCommentTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Issue Comment Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onIssueCommentTriggerRenderer.getTriggerProps(ctx).title).toBe("My Issue Comment Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On Issue Comment" }),
      lastEvent: buildEvent(),
    };
    expect(onIssueCommentTriggerRenderer.getTriggerProps(ctx).title).toBe("On Issue Comment");
  });

  it("includes project metadata and content filter", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { id: 1, name: "group/example" } },
        configuration: { contentFilter: "/sp-investigate" },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onIssueCommentTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata?.find((m) => String(m.label).includes("group/example"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("/sp-investigate"))).toBeDefined();
  });

  it("omits metadata when project and content filter are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onIssueCommentTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([]);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({ data: buildEventData() }),
    };
    const props = onIssueCommentTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData!.title).toBe("#8 - Login page throws 500 on invalid credentials");
    expect(props.lastEventData!.state).toBe("triggered");
  });

  it("omits lastEventData when lastEvent is missing", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: undefined,
    };
    expect(onIssueCommentTriggerRenderer.getTriggerProps(ctx).lastEventData).toBeUndefined();
  });
});
