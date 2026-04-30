import { describe, expect, it } from "vitest";

import { onComputeInstanceCreatedTriggerRenderer } from "./on_compute_instance_created";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";

const CREATED_AT = new Date("2026-01-01T00:00:00Z").toISOString();

function buildEvent(data?: Record<string, unknown>) {
  return {
    id: "evt-1",
    createdAt: CREATED_AT,
    nodeId: "node-1",
    type: "oci.onComputeInstanceCreated",
    data: data ?? {},
  };
}

function buildOciEventData(overrides?: {
  resourceName?: string;
  resourceId?: string;
  compartmentId?: string;
  compartmentName?: string;
  availabilityDomain?: string;
  shape?: string;
  eventTime?: string;
}) {
  return {
    eventType: "com.oraclecloud.computeapi.launchinstance.end",
    eventTime: overrides?.eventTime ?? CREATED_AT,
    data: {
      resourceName: overrides?.resourceName ?? "my-instance",
      resourceId: overrides?.resourceId ?? "ocid1.instance.oc1.eu-frankfurt-1.aaaaaaaExample",
      compartmentId: overrides?.compartmentId ?? "ocid1.tenancy.oc1..aaaaaaaExample",
      compartmentName: overrides?.compartmentName ?? "root",
      availabilityDomain: overrides?.availabilityDomain ?? "EXAMPLE:EU-FRANKFURT-1-AD-1",
      additionalDetails: {
        shape: overrides?.shape ?? "VM.Standard.E4.Flex",
      },
    },
  };
}

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "",
    componentName: "oci.onComputeInstanceCreated",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildTriggerRendererCtx(overrides?: {
  nodeName?: string;
  lastEvent?: ReturnType<typeof buildEvent>;
}): TriggerRendererContext {
  return {
    node: buildNode({ name: overrides?.nodeName ?? "" }),
    definition: {
      name: "oci.onComputeInstanceCreated",
      label: "On Compute Instance Created",
      description: "",
      icon: "oci",
      color: "red",
    },
    lastEvent: overrides?.lastEvent ?? buildEvent(buildOciEventData()),
  };
}

// ── getTitleAndSubtitle ────────────────────────────────────────────────

describe("onComputeInstanceCreatedTriggerRenderer.getTitleAndSubtitle", () => {
  it("returns fixed title and instance name as subtitle", () => {
    const ctx: TriggerEventContext = { event: buildEvent(buildOciEventData({ resourceName: "my-vm" })) };
    const { title, subtitle } = onComputeInstanceCreatedTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(title).toBe("Compute instance created");
    expect(subtitle).toBe("my-vm");
  });

  it("returns empty subtitle when resourceName is absent", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ eventType: "com.oraclecloud.computeapi.launchinstance.end", data: {} }),
    };
    const { subtitle } = onComputeInstanceCreatedTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(subtitle).toBe("");
  });

  it("does not throw when event is undefined", () => {
    const ctx: TriggerEventContext = { event: undefined };
    expect(() => onComputeInstanceCreatedTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ─────────────────────────────────────────────────

describe("onComputeInstanceCreatedTriggerRenderer.getRootEventValues", () => {
  it("includes all fields when data is complete", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent(
        buildOciEventData({
          resourceName: "my-instance",
          compartmentName: "root",
          availabilityDomain: "EXAMPLE:EU-FRANKFURT-1-AD-1",
          shape: "VM.Standard.E4.Flex",
        }),
      ),
    };
    const values = onComputeInstanceCreatedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Instance Name"]).toBe("my-instance");
    expect(values["Compartment"]).toBe("root");
    expect(values["Availability Domain"]).toBe("EXAMPLE:EU-FRANKFURT-1-AD-1");
    expect(values["Shape"]).toBe("VM.Standard.E4.Flex");
  });

  it("uses event.createdAt for Triggered At", () => {
    const ctx: TriggerEventContext = { event: buildEvent(buildOciEventData()) };
    const values = onComputeInstanceCreatedTriggerRenderer.getRootEventValues(ctx);
    expect(new Date(values["Triggered At"]).getTime()).toBe(new Date(CREATED_AT).getTime());
  });

  it("falls back to envelope.eventTime when event.createdAt is absent", () => {
    const eventTime = "2026-01-02T10:00:00Z";
    const ociData = buildOciEventData({ eventTime });
    const ctx: TriggerEventContext = {
      event: { ...buildEvent(ociData), createdAt: undefined as unknown as string },
    };
    const values = onComputeInstanceCreatedTriggerRenderer.getRootEventValues(ctx);
    expect(new Date(values["Triggered At"]).getTime()).toBe(new Date(eventTime).getTime());
  });

  it("omits fields whose values are missing", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ eventType: "com.oraclecloud.computeapi.launchinstance.end", data: {} }),
    };
    const values = onComputeInstanceCreatedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Instance Name"]).toBeUndefined();
    expect(values["Shape"]).toBeUndefined();
    expect(values["Compartment"]).toBeUndefined();
  });

  it("does not throw when event is undefined", () => {
    const ctx: TriggerEventContext = { event: undefined };
    expect(() => onComputeInstanceCreatedTriggerRenderer.getRootEventValues(ctx)).not.toThrow();
  });
});

// ── getTriggerProps ────────────────────────────────────────────────────

describe("onComputeInstanceCreatedTriggerRenderer.getTriggerProps", () => {
  it("uses node.name as title when set", () => {
    const ctx = buildTriggerRendererCtx({ nodeName: "My OCI Trigger" });
    const props = onComputeInstanceCreatedTriggerRenderer.getTriggerProps(ctx);
    expect(props.title).toBe("My OCI Trigger");
  });

  it("falls back to definition.label when node.name is empty", () => {
    const ctx = buildTriggerRendererCtx({ nodeName: "" });
    const props = onComputeInstanceCreatedTriggerRenderer.getTriggerProps(ctx);
    expect(props.title).toBe("On Compute Instance Created");
  });

  it("sets lastEventData.title to instance name when present", () => {
    const ctx = buildTriggerRendererCtx({
      lastEvent: buildEvent(buildOciEventData({ resourceName: "prod-instance" })),
    });
    const props = onComputeInstanceCreatedTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData?.title).toBe("prod-instance");
  });

  it("sets lastEventData.title to fallback when resourceName is absent", () => {
    const ctx = buildTriggerRendererCtx({
      lastEvent: buildEvent({ data: {} }),
    });
    const props = onComputeInstanceCreatedTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData?.title).toBe("Compute instance created");
  });

  it("sets lastEventData.state to triggered", () => {
    const ctx = buildTriggerRendererCtx();
    const props = onComputeInstanceCreatedTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData?.state).toBe("triggered");
  });

  it("sets receivedAt from lastEvent.createdAt", () => {
    const ctx = buildTriggerRendererCtx();
    const props = onComputeInstanceCreatedTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData?.receivedAt).toEqual(new Date(CREATED_AT));
  });
});
