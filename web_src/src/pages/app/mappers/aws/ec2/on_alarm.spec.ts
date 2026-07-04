import { describe, expect, it } from "vitest";
import { onEc2AlarmTriggerRenderer } from "./on_alarm";
import { triggerRenderers } from "../index";
import type { TriggerEventContext, TriggerRendererContext, NodeInfo } from "../../types";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildEvent(overrides?: Record<string, unknown>): TriggerEventContext["event"] {
  return {
    id: "evt-1",
    createdAt: new Date("2026-05-21T12:01:00Z").toISOString(),
    data: {
      region: "us-east-1",
      account: "123456789012",
      detail: {
        alarmName: "HighCPU",
        state: { value: "ALARM" },
        previousState: { value: "OK" },
      },
    },
    ...overrides,
  } as TriggerEventContext["event"];
}

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "EC2 Alarm Trigger",
    componentName: "aws.ec2.onAlarm",
    isCollapsed: false,
    configuration: {
      region: "us-east-1",
      instance: "i-abc123",
      state: "ALARM",
      alarm: "HighCPU",
    },
    metadata: {
      instanceId: "i-abc123",
      instanceName: "web-server-1",
      region: "us-east-1",
    },
    ...overrides,
  };
}

function buildRendererContext(overrides?: Partial<TriggerRendererContext>): TriggerRendererContext {
  return {
    node: buildNode(),
    definition: {
      name: "aws.ec2.onAlarm",
      label: "EC2 • On Alarm",
      description: "",
      icon: "aws",
      color: "gray",
    },
    lastEvent: undefined,
    ...overrides,
  } as TriggerRendererContext;
}

// ── getTitleAndSubtitle ───────────────────────────────────────────────────────

describe("onEc2AlarmTriggerRenderer.getTitleAndSubtitle", () => {
  it("formats alarm name with state transition", () => {
    const { title } = onEc2AlarmTriggerRenderer.getTitleAndSubtitle({
      event: buildEvent(),
    });
    expect(title).toBe("HighCPU \u2014 OK \u2192 ALARM");
  });

  it("falls back to alarm name when previous state is missing", () => {
    const { title } = onEc2AlarmTriggerRenderer.getTitleAndSubtitle({
      event: buildEvent({
        data: {
          detail: { alarmName: "HighCPU", state: { value: "ALARM" } },
        },
      }),
    });
    expect(title).toBe("HighCPU");
  });

  it("falls back to default title when detail is empty", () => {
    const { title } = onEc2AlarmTriggerRenderer.getTitleAndSubtitle({
      event: buildEvent({ data: {} }),
    });
    expect(title).toBe("EC2 alarm state change");
  });
});

// ── getRootEventValues ────────────────────────────────────────────────────────

describe("onEc2AlarmTriggerRenderer.getRootEventValues", () => {
  it("extracts all event fields", () => {
    const values = onEc2AlarmTriggerRenderer.getRootEventValues({ event: buildEvent() });
    expect(values["Alarm"]).toBe("HighCPU");
    expect(values["State"]).toBe("ALARM");
    expect(values["Previous State"]).toBe("OK");
    expect(values["Region"]).toBe("us-east-1");
    expect(values["Account"]).toBe("123456789012");
  });

  it("includes Triggered At timestamp as first field", () => {
    const values = onEc2AlarmTriggerRenderer.getRootEventValues({ event: buildEvent() });
    expect(values["Triggered At"]).not.toBe("-");
    const keys = Object.keys(values);
    expect(keys[0]).toBe("Triggered At");
  });

  it("returns dash for Triggered At when no event createdAt", () => {
    const values = onEc2AlarmTriggerRenderer.getRootEventValues({
      event: { ...buildEvent(), createdAt: undefined } as unknown as TriggerEventContext["event"],
    });
    expect(values["Triggered At"]).toBe("-");
  });

  it("returns dashes for missing fields", () => {
    const values = onEc2AlarmTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Alarm"]).toBe("-");
    expect(values["State"]).toBe("-");
    expect(values["Region"]).toBe("-");
  });
});

// ── getTriggerProps ───────────────────────────────────────────────────────────

describe("onEc2AlarmTriggerRenderer.getTriggerProps", () => {
  it("uses node name as title", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(buildRendererContext());
    expect(props.title).toBe("EC2 Alarm Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(buildRendererContext({ node: buildNode({ name: "" }) }));
    expect(props.title).toBe("EC2 • On Alarm");
  });

  it("includes instance name in metadata", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(buildRendererContext());
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("web-server-1");
  });

  it("includes region in metadata when no alarm name set", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(
      buildRendererContext({
        node: buildNode({ configuration: { region: "us-east-1", instance: "i-abc123", state: "ALARM" } }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("us-east-1");
  });

  it("includes state filter in metadata", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(buildRendererContext());
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("ALARM");
  });

  it("includes alarm name in metadata when configured", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(buildRendererContext());
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("HighCPU");
  });

  it("does not include alarm name when not configured", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(
      buildRendererContext({
        node: buildNode({ configuration: { region: "us-east-1", instance: "i-abc123", state: "ALARM" } }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).not.toContain("HighCPU");
  });

  it("does not throw with empty configuration", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(
      buildRendererContext({ node: buildNode({ configuration: {}, metadata: {} }) }),
    );
    expect(props.metadata).toEqual([]);
  });

  it("populates lastEventData when lastEvent is present", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(buildRendererContext({ lastEvent: buildEvent() }));
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData?.state).toBe("triggered");
    expect(props.lastEventData?.eventId).toBe("evt-1");
  });

  it("omits lastEventData when no lastEvent", () => {
    const props = onEc2AlarmTriggerRenderer.getTriggerProps(buildRendererContext({ lastEvent: undefined }));
    expect(props.lastEventData).toBeUndefined();
  });
});

// ── triggerRenderers registration ─────────────────────────────────────────────

describe("triggerRenderers['ec2.onAlarm']", () => {
  it("is registered in the global trigger renderers map", () => {
    expect(triggerRenderers["ec2.onAlarm"]).toBe(onEc2AlarmTriggerRenderer);
  });
});
