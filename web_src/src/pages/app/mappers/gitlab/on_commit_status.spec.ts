import { describe, expect, it } from "vitest";

import { onCommitStatusTriggerRenderer } from "./on_commit_status";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext, ComponentDefinition, EventInfo } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Commit Status",
    componentName: "gitlab.onCommitStatus",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onCommitStatus",
    label: "On Commit Status",
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
    type: "gitlab.commitStatusChanged",
    ...overrides,
  };
}

function buildEventData() {
  return {
    object_kind: "pipeline",
    object_attributes: {
      id: 12345,
      iid: 321,
      ref: "main",
      sha: "f4f6c5a0d2e5ad34be4c17c3f166f4d2ff8b0a55",
      status: "success",
      source: "external",
      url: "https://gitlab.com/group/example-project/-/pipelines/12345",
    },
    builds: [
      { id: 998877, stage: "external", name: "security-scan", status: "success" },
      { id: 998876, stage: "test", name: "unit-tests", status: "success" },
    ],
    commit: {
      id: "f4f6c5a0d2e5ad34be4c17c3f166f4d2ff8b0a55",
      title: "Cache dependencies between CI jobs",
      url: "https://gitlab.com/group/example-project/-/commit/f4f6c5a0d2e5ad34be4c17c3f166f4d2ff8b0a55",
    },
    project: {
      id: 987,
      name: "example-project",
      path_with_namespace: "group/example-project",
      web_url: "https://gitlab.com/group/example-project",
    },
  };
}

// ── getTitleAndSubtitle ─────────────────────────────────────────────

describe("onCommitStatusTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses the short commit SHA in the title", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(onCommitStatusTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Commit f4f6c5a0");
  });

  it("falls back to the commit object when object_attributes has no SHA", () => {
    const data = { ...buildEventData(), object_attributes: { status: "failed" } };
    const ctx: TriggerEventContext = { event: buildEvent({ data }) };
    expect(onCommitStatusTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Commit f4f6c5a0");
  });

  it("falls back to a default title when no SHA is present", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: {} }) };
    expect(onCommitStatusTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Commit Status");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: undefined }) };
    expect(() => onCommitStatusTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ──────────────────────────────────────────────

describe("onCommitStatusTriggerRenderer.getRootEventValues", () => {
  it("extracts commit status details from event data", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onCommitStatusTriggerRenderer.getRootEventValues(ctx);
    expect(values["Status"]).toBe("success");
    expect(values["Status Names"]).toBe("security-scan, unit-tests");
    expect(values["Ref"]).toBe("main");
    expect(values["Commit"]).toBe("f4f6c5a0");
    expect(values["Pipeline URL"]).toBe("https://gitlab.com/group/example-project/-/pipelines/12345");
  });

  it("includes received at as the first value", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onCommitStatusTriggerRenderer.getRootEventValues(ctx);
    expect(Object.keys(values)[0]).toBe("Received At");
    expect(values["Received At"]).not.toBe("-");
  });

  it("returns at most 6 values", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(Object.keys(onCommitStatusTriggerRenderer.getRootEventValues(ctx)).length).toBeLessThanOrEqual(6);
  });

  it("skips builds without a name", () => {
    const data = { ...buildEventData(), builds: [{ id: 1, status: "success" }, { name: "lint" }] };
    const ctx: TriggerEventContext = { event: buildEvent({ data }) };
    expect(onCommitStatusTriggerRenderer.getRootEventValues(ctx)["Status Names"]).toBe("lint");
  });

  it("returns placeholders when event data is missing", () => {
    const values = onCommitStatusTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Status"]).toBe("-");
    expect(values["Status Names"]).toBe("-");
    expect(values["Ref"]).toBe("-");
    expect(values["Commit"]).toBe("-");
    expect(values["Pipeline URL"]).toBe("-");
  });
});

// ── getTriggerProps ─────────────────────────────────────────────────

describe("onCommitStatusTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Commit Status Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onCommitStatusTriggerRenderer.getTriggerProps(ctx).title).toBe("My Commit Status Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On Commit Status" }),
      lastEvent: buildEvent(),
    };
    expect(onCommitStatusTriggerRenderer.getTriggerProps(ctx).title).toBe("On Commit Status");
  });

  it("includes project metadata and configured filters", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { id: 987, name: "group/example-project" } },
        configuration: {
          statuses: ["success", "failed"],
          names: [{ type: "equals", value: "security-scan" }],
        },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onCommitStatusTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata?.find((m) => String(m.label).includes("group/example-project"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("success, failed"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("security-scan"))).toBeDefined();
  });

  it("omits metadata when project and filters are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onCommitStatusTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([]);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({ data: buildEventData() }),
    };
    const props = onCommitStatusTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData!.title).toBe("Commit f4f6c5a0");
    expect(props.lastEventData!.state).toBe("triggered");
  });

  it("omits lastEventData when lastEvent is missing", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: undefined,
    };
    expect(onCommitStatusTriggerRenderer.getTriggerProps(ctx).lastEventData).toBeUndefined();
  });
});
