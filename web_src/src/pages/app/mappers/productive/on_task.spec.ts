import { describe, expect, it } from "vitest";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onTaskTriggerRenderer } from "./on_task";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-03-26T19:29:35Z").toISOString(),
    nodeId: "node-1",
    type: "productive.task",
    data,
  };
}

const taskEvent = {
  meta: { event: "task.created" },
  data: {
    id: "98765",
    type: "tasks",
    attributes: {
      title: "Fix payment retries",
      description: "Retries fail silently after the third attempt.",
    },
  },
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Task",
    componentName: "productive.onTask",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildTriggerContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastEvent?: EventInfo;
  definition?: Partial<ComponentDefinition>;
}): TriggerRendererContext {
  return {
    node: buildNode(overrides?.node),
    definition: {
      name: "productive.onTask",
      label: "On Task",
      description: "",
      icon: "productive",
      color: "indigo",
      ...overrides?.definition,
    },
    lastEvent: overrides?.lastEvent,
  } as TriggerRendererContext;
}

describe("onTaskTriggerRenderer.getTitleAndSubtitle", () => {
  it("builds the title from the task attributes", () => {
    const context: TriggerEventContext = { event: event(taskEvent) };
    expect(onTaskTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Fix payment retries");
  });

  it("falls back to a generic title when the payload is empty", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onTaskTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Task");
  });

  it("does not throw when there is no event", () => {
    expect(() => onTaskTriggerRenderer.getTitleAndSubtitle({ event: undefined })).not.toThrow();
  });
});

describe("onTaskTriggerRenderer.getRootEventValues", () => {
  it("maps the task fields, including the description", () => {
    const values = onTaskTriggerRenderer.getRootEventValues({ event: event(taskEvent) });

    expect(values["Received At"]).toBeDefined();
    expect(values["Task"]).toBe("98765");
    expect(values["Title"]).toBe("Fix payment retries");
    expect(values["Action"]).toBe("Created");
    expect(values["Description"]).toBe("Retries fail silently after the third attempt.");
  });

  it("humanises the updated event", () => {
    const values = onTaskTriggerRenderer.getRootEventValues({
      event: event({ ...taskEvent, meta: { event: "task.updated" } }),
    });

    expect(values["Action"]).toBe("Updated");
  });

  it("passes an unknown event through unchanged", () => {
    const values = onTaskTriggerRenderer.getRootEventValues({
      event: event({ ...taskEvent, meta: { event: "task.archived" } }),
    });

    expect(values["Action"]).toBe("task.archived");
  });

  it("falls back to dashes when the payload is empty", () => {
    const values = onTaskTriggerRenderer.getRootEventValues({ event: event({}) });

    expect(values["Task"]).toBe("-");
    expect(values["Title"]).toBe("-");
    expect(values["Description"]).toBe("-");
  });

  it("does not throw when there is no event", () => {
    expect(() => onTaskTriggerRenderer.getRootEventValues({ event: undefined })).not.toThrow();
  });
});

describe("onTaskTriggerRenderer.getTriggerProps", () => {
  it("renders the project and selected actions", () => {
    const props = onTaskTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { project: "1", actions: ["created", "updated"] },
          metadata: { project: { id: "1", name: "Payments" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "folder", label: "Payments" });
    expect(props.metadata?.[1]).toEqual({ icon: "funnel", label: "Created, Updated" });
  });

  it("falls back to the configured project when metadata is missing", () => {
    const props = onTaskTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { project: "1", actions: ["created"] } } }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "folder", label: "1" });
  });

  it("omits the actions badge when none are configured", () => {
    const props = onTaskTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { project: "1" } } }),
    );

    expect((props.metadata || []).map((item) => item.icon)).not.toContain("funnel");
  });

  it("surfaces the last event when one exists", () => {
    const props = onTaskTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: { configuration: { project: "1", actions: ["created"] } },
        lastEvent: event(taskEvent),
      }),
    );

    expect(props.lastEventData?.title).toBe("Fix payment retries");
    expect(props.lastEventData?.state).toBe("triggered");
  });

  it("does not throw when configuration and metadata are undefined", () => {
    expect(() =>
      onTaskTriggerRenderer.getTriggerProps(
        buildTriggerContext({ node: { configuration: undefined, metadata: undefined } }),
      ),
    ).not.toThrow();
  });
});
