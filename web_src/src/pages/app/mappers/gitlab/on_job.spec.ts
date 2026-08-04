import { describe, expect, it } from "vitest";

import { onJobTriggerRenderer } from "./on_job";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext, ComponentDefinition, EventInfo } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Job",
    componentName: "gitlab.onJob",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onJob",
    label: "On Job",
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
    type: "gitlab.job",
    ...overrides,
  };
}

function buildEventData() {
  return {
    object_kind: "build",
    ref: "main",
    sha: "f4f6c5a0d2e5ad34be4c17c3f166f4d2ff8b0a55",
    build_id: 998876,
    build_name: "unit-tests",
    build_stage: "test",
    build_status: "success",
    pipeline_id: 12345,
    project: {
      id: 987,
      name: "example-project",
      path_with_namespace: "group/example-project",
      web_url: "https://gitlab.com/group/example-project",
    },
  };
}

// ── getTitleAndSubtitle ─────────────────────────────────────────────

describe("onJobTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses the job name in the title", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(onJobTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Job: unit-tests");
  });

  it("falls back to a default title when the job name is missing", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: {} }) };
    expect(onJobTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Job");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: undefined }) };
    expect(() => onJobTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ──────────────────────────────────────────────

describe("onJobTriggerRenderer.getRootEventValues", () => {
  it("extracts job details from event data", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onJobTriggerRenderer.getRootEventValues(ctx);
    expect(values["Job"]).toBe("unit-tests");
    expect(values["Status"]).toBe("success");
    expect(values["Stage"]).toBe("test");
    expect(values["Ref"]).toBe("main");
    expect(values["Job URL"]).toBe("https://gitlab.com/group/example-project/-/jobs/998876");
  });

  it("includes received at as the first value", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    const values = onJobTriggerRenderer.getRootEventValues(ctx);
    expect(Object.keys(values)[0]).toBe("Received At");
    expect(values["Received At"]).not.toBe("-");
  });

  it("returns at most 6 values", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: buildEventData() }) };
    expect(Object.keys(onJobTriggerRenderer.getRootEventValues(ctx)).length).toBeLessThanOrEqual(6);
  });

  it("omits the job URL when the project web_url is missing", () => {
    const data = { ...buildEventData(), project: undefined };
    const ctx: TriggerEventContext = { event: buildEvent({ data }) };
    expect(onJobTriggerRenderer.getRootEventValues(ctx)["Job URL"]).toBe("-");
  });

  it("returns placeholders when event data is missing", () => {
    const values = onJobTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Job"]).toBe("-");
    expect(values["Status"]).toBe("-");
    expect(values["Stage"]).toBe("-");
    expect(values["Ref"]).toBe("-");
    expect(values["Job URL"]).toBe("-");
  });
});

// ── getTriggerProps ─────────────────────────────────────────────────

describe("onJobTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Job Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onJobTriggerRenderer.getTriggerProps(ctx).title).toBe("My Job Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On Job" }),
      lastEvent: buildEvent(),
    };
    expect(onJobTriggerRenderer.getTriggerProps(ctx).title).toBe("On Job");
  });

  it("includes project metadata and configured filters", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { id: 987, name: "group/example-project" } },
        configuration: {
          statuses: ["success", "failed"],
          names: [{ type: "equals", value: "deploy" }],
          refs: [{ type: "matches", value: "release/.*" }],
        },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onJobTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata?.find((m) => String(m.label).includes("group/example-project"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("success, failed"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("deploy"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("release/.*"))).toBeDefined();
  });

  it("omits metadata when project and filters are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onJobTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([]);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({ data: buildEventData() }),
    };
    const props = onJobTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData!.title).toBe("Job: unit-tests");
    expect(props.lastEventData!.state).toBe("triggered");
  });

  it("omits lastEventData when lastEvent is missing", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: undefined,
    };
    expect(onJobTriggerRenderer.getTriggerProps(ctx).lastEventData).toBeUndefined();
  });
});
