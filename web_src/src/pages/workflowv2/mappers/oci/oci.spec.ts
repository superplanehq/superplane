import { describe, expect, it } from "vitest";

import { deleteInstanceMapper } from "./delete_instance";
import { eventStateRegistry } from "./index";
import { getInstanceMapper } from "./get_instance";
import { manageInstancePowerMapper } from "./manage_instance_power";
import { onInstanceStateChangeTriggerRenderer } from "./on_instance_state_change";
import { updateInstanceMapper } from "./update_instance";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ComponentDefinition,
  EventInfo,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  TriggerRendererContext,
} from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "oci.getInstance",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "oci.result",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: "2026-04-22T20:32:00.000Z",
    updatedAt: "2026-04-22T20:32:10.000Z",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  };
}

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

function buildComponentContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastExecutions?: ExecutionInfo[];
  componentDefinition?: Partial<ComponentDefinition>;
}): ComponentBaseContext {
  const node = buildNode(overrides?.node);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "oci.getInstance",
      label: "Get Instance",
      description: "",
      icon: "oci",
      color: "red",
      ...overrides?.componentDefinition,
    },
    lastExecutions: overrides?.lastExecutions ?? [],
    currentUser: undefined,
    actions: { invokeNodeExecutionAction: async () => {} },
  };
}

const fullInstanceOutput = {
  instanceId: "ocid1.instance.oc1.eu-frankfurt-1.example",
  displayName: "test-instance",
  lifecycleState: "RUNNING",
  shape: "VM.Standard.E2.1.Micro",
  availabilityDomain: "XXXX:eu-frankfurt-1-AD-1",
  compartmentId: "ocid1.tenancy.oc1..example",
  region: "eu-frankfurt-1",
  timeCreated: "2026-04-22T20:31:25.145Z",
};

const componentCases: Array<{
  name: string;
  mapper: ComponentBaseMapper;
  config: Record<string, unknown>;
  metadata: Array<{ icon: string; label: string }>;
  minimalMetadata: Array<{ icon: string; label: string }>;
  emptyFields: string[];
  fullAssertions: Record<string, string>;
}> = [
  {
    name: "getInstance",
    mapper: getInstanceMapper,
    config: { instanceId: "ocid1.instance.oc1.eu-frankfurt-1.example" },
    metadata: [{ icon: "server", label: "ocid1.instance.oc1.eu-frankfurt-1.example" }],
    minimalMetadata: [],
    emptyFields: ["Instance ID", "Display Name", "State", "Shape", "Region"],
    fullAssertions: {
      "Instance ID": fullInstanceOutput.instanceId,
      "Display Name": fullInstanceOutput.displayName,
      State: fullInstanceOutput.lifecycleState,
      Shape: fullInstanceOutput.shape,
      Region: fullInstanceOutput.region,
    },
  },
  {
    name: "updateInstance",
    mapper: updateInstanceMapper,
    config: { instanceId: "ocid1.instance.oc1.eu-frankfurt-1.example", displayName: "renamed" },
    metadata: [
      { icon: "server", label: "ocid1.instance.oc1.eu-frankfurt-1.example" },
      { icon: "tag", label: "renamed" },
    ],
    minimalMetadata: [],
    emptyFields: ["Instance ID", "Display Name", "State", "Shape", "Region"],
    fullAssertions: {
      "Instance ID": fullInstanceOutput.instanceId,
      "Display Name": fullInstanceOutput.displayName,
      State: fullInstanceOutput.lifecycleState,
      Shape: fullInstanceOutput.shape,
      Region: fullInstanceOutput.region,
    },
  },
  {
    name: "manageInstancePower",
    mapper: manageInstancePowerMapper,
    config: { instanceId: "ocid1.instance.oc1.eu-frankfurt-1.example", action: "STOP" },
    metadata: [
      { icon: "server", label: "ocid1.instance.oc1.eu-frankfurt-1.example" },
      { icon: "zap", label: "STOP" },
    ],
    minimalMetadata: [],
    emptyFields: ["Instance ID", "Display Name", "State", "Region"],
    fullAssertions: {
      Action: "STOP",
      "Instance ID": fullInstanceOutput.instanceId,
      "Display Name": fullInstanceOutput.displayName,
      State: fullInstanceOutput.lifecycleState,
      Region: fullInstanceOutput.region,
    },
  },
  {
    name: "deleteInstance",
    mapper: deleteInstanceMapper,
    config: { instanceId: "ocid1.instance.oc1.eu-frankfurt-1.example", preserveBootVolume: true },
    metadata: [
      { icon: "trash-2", label: "ocid1.instance.oc1.eu-frankfurt-1.example" },
      { icon: "archive", label: "Preserve boot volume" },
    ],
    minimalMetadata: [],
    emptyFields: ["Instance ID", "State"],
    fullAssertions: {
      "Instance ID": fullInstanceOutput.instanceId,
      State: "TERMINATED",
      "Preserve Boot Volume": "Yes",
    },
  },
];

for (const testCase of componentCases) {
  describe(`${testCase.name}Mapper.getExecutionDetails`, () => {
    it("does not throw when outputs is undefined", () => {
      const ctx = buildDetailsCtx({
        node: { componentName: `oci.${testCase.name}`, configuration: testCase.config },
        execution: { outputs: undefined },
      });
      expect(() => testCase.mapper.getExecutionDetails(ctx)).not.toThrow();
    });

    it("does not throw when default array is empty", () => {
      const ctx = buildDetailsCtx({
        node: { componentName: `oci.${testCase.name}`, configuration: testCase.config },
        execution: { outputs: { default: [] } },
      });
      expect(() => testCase.mapper.getExecutionDetails(ctx)).not.toThrow();
    });

    it("returns dashes for expected instance fields when data is empty", () => {
      const ctx = buildDetailsCtx({
        node: { componentName: `oci.${testCase.name}`, configuration: {} },
        execution: { outputs: { default: [buildOutput({})] } },
      });
      const details = testCase.mapper.getExecutionDetails(ctx);

      for (const field of testCase.emptyFields) {
        expect(details[field]).toBe("-");
      }
    });

    it("extracts fields from a full output payload", () => {
      const payload =
        testCase.name === "deleteInstance"
          ? { instanceId: fullInstanceOutput.instanceId, lifecycleState: "TERMINATED" }
          : fullInstanceOutput;
      const ctx = buildDetailsCtx({
        node: { componentName: `oci.${testCase.name}`, configuration: testCase.config },
        execution: { outputs: { default: [buildOutput(payload)] } },
      });
      const details = testCase.mapper.getExecutionDetails(ctx);

      for (const [field, value] of Object.entries(testCase.fullAssertions)) {
        expect(details[field]).toBe(value);
      }
    });

    it("includes Executed At when createdAt is present", () => {
      const ctx = buildDetailsCtx({
        node: { componentName: `oci.${testCase.name}`, configuration: testCase.config },
        execution: { outputs: { default: [buildOutput(fullInstanceOutput)] } },
      });
      expect(testCase.mapper.getExecutionDetails(ctx)["Executed At"]).not.toBe("-");
    });
  });

  describe(`${testCase.name}Mapper.props`, () => {
    it("returns expected metadata when config fields are set", () => {
      const props = testCase.mapper.props(
        buildComponentContext({
          node: { componentName: `oci.${testCase.name}`, configuration: testCase.config },
          componentDefinition: { name: `oci.${testCase.name}` },
        }),
      );

      expect(props.metadata).toEqual(testCase.metadata);
    });

    it("returns empty or minimal metadata when config fields are absent", () => {
      const props = testCase.mapper.props(
        buildComponentContext({
          node: { componentName: `oci.${testCase.name}`, configuration: {} },
          componentDefinition: { name: `oci.${testCase.name}` },
        }),
      );

      expect(props.metadata).toEqual(testCase.minimalMetadata);
    });
  });
}

function buildEvent(eventType: string): NonNullable<EventInfo> {
  return {
    id: "event-1",
    createdAt: "2026-04-22T20:34:54.000Z",
    nodeId: "trigger-1",
    type: "oci.onInstanceStateChange",
    data: {
      eventType,
      eventTime: "2026-04-22T20:34:54Z",
      data: {
        resourceName: "test-instance",
        resourceId: "ocid1.instance.oc1.eu-frankfurt-1.example",
        compartmentId: "ocid1.tenancy.oc1..example",
        compartmentName: "root",
        availabilityDomain: "XXXX:eu-frankfurt-1-AD-1",
        additionalDetails: {
          shape: "VM.Standard.E2.1.Micro",
        },
      },
    },
  };
}

function buildTriggerContext(lastEvent?: EventInfo): TriggerRendererContext {
  return {
    node: buildNode({ componentName: "oci.onInstanceStateChange", name: "State Change" }),
    definition: {
      name: "oci.onInstanceStateChange",
      label: "On Instance State Change",
      description: "",
      icon: "oci",
      color: "red",
    },
    lastEvent,
  };
}

describe("onInstanceStateChangeTriggerRenderer", () => {
  const labels: Record<string, string> = {
    "com.oraclecloud.computeapi.startinstance.end": "Instance started",
    "com.oraclecloud.computeapi.stopinstance.end": "Instance stopped",
    "com.oraclecloud.computeapi.terminateinstance.end": "Instance terminated",
    "com.oraclecloud.computeapi.resetinstance.end": "Instance reset",
    "com.oraclecloud.computeapi.softstopinstance.end": "Instance soft-stopped",
    "com.oraclecloud.computeapi.softresetinstance.end": "Instance soft-reset",
  };

  for (const [eventType, label] of Object.entries(labels)) {
    it(`returns ${label} for ${eventType}`, () => {
      expect(onInstanceStateChangeTriggerRenderer.getTitleAndSubtitle({ event: buildEvent(eventType) }).title).toBe(
        label,
      );
    });
  }

  it("extracts all root event values", () => {
    const details = onInstanceStateChangeTriggerRenderer.getRootEventValues({
      event: buildEvent("com.oraclecloud.computeapi.stopinstance.end"),
    });

    expect(details["Triggered At"]).toBeDefined();
    expect(details["Instance Name"]).toBe("test-instance");
    expect(details["Instance ID"]).toBe("ocid1.instance.oc1.eu-frankfurt-1.example");
    expect(details["Shape"]).toBe("VM.Standard.E2.1.Micro");
    expect(details["Availability Domain"]).toBe("XXXX:eu-frankfurt-1-AD-1");
    expect(details["Compartment"]).toBe("root");
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const props = onInstanceStateChangeTriggerRenderer.getTriggerProps(
      buildTriggerContext(buildEvent("com.oraclecloud.computeapi.stopinstance.end")),
    );

    expect(props.lastEventData).toMatchObject({
      title: "Instance stopped",
      subtitle: "test-instance",
      state: "triggered",
      eventId: "event-1",
    });
  });

  it("omits lastEventData when lastEvent is absent", () => {
    const props = onInstanceStateChangeTriggerRenderer.getTriggerProps(buildTriggerContext(undefined));
    expect(props.lastEventData).toBeUndefined();
  });
});

describe("eventStateRegistry", () => {
  it("maps finished success to fetched for getInstance", () => {
    expect(eventStateRegistry.getInstance.getState(buildExecution())).toBe("fetched");
  });

  it("maps finished success to updated for updateInstance", () => {
    expect(eventStateRegistry.updateInstance.getState(buildExecution())).toBe("updated");
  });

  it("maps finished success to completed for manageInstancePower", () => {
    expect(eventStateRegistry.manageInstancePower.getState(buildExecution())).toBe("completed");
  });

  it("maps finished success to deleted for deleteInstance", () => {
    expect(eventStateRegistry.deleteInstance.getState(buildExecution())).toBe("deleted");
  });

  it("maps in-progress executions to running", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED",
      resultReason: "RESULT_REASON_UNSPECIFIED",
    });

    expect(eventStateRegistry.getInstance.getState(execution)).toBe("running");
    expect(eventStateRegistry.updateInstance.getState(execution)).toBe("running");
    expect(eventStateRegistry.manageInstancePower.getState(execution)).toBe("running");
    expect(eventStateRegistry.deleteInstance.getState(execution)).toBe("running");
  });
});
