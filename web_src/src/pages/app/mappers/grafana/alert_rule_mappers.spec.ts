import { describe, expect, it } from "vitest";

import { createAlertRuleMapper } from "./create_alert_rule";
import { deleteAlertRuleMapper } from "./delete_alert_rule";
import { getAlertRuleMapper } from "./get_alert_rule";
import { listAlertRulesMapper } from "./list_alert_rules";
import { updateAlertRuleMapper } from "./update_alert_rule";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function buildNode(componentName: string, overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Alert Rule Mapper",
    componentName,
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
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

function buildComponentContext(componentName: string, nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  const node = buildNode(componentName, nodeOverrides);

  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: componentName,
      label: componentName,
      description: "",
      icon: "bolt",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
  };
}

function buildExecutionContext(
  componentName: string,
  overrides?: { node?: Partial<NodeInfo>; execution?: Partial<ExecutionInfo> },
): ExecutionDetailsContext {
  const node = buildNode(componentName, overrides?.node);

  return {
    nodes: [node],
    node,
    execution: buildExecution(overrides?.execution),
  };
}

describe("Grafana alert rule mappers", () => {
  it("createAlertRuleMapper renders title, rule group, and paused metadata", () => {
    const props = createAlertRuleMapper.props(
      buildComponentContext("grafana.createAlertRule", {
        configuration: {
          title: "High error rate",
          ruleGroup: "service-health",
          isPaused: true,
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ label: "High error rate" }),
        expect.objectContaining({ label: "Paused" }),
        expect.objectContaining({ label: "service-health" }),
      ]),
    );
  });

  it("getAlertRuleMapper uses configuration.alertRule in metadata", () => {
    const props = getAlertRuleMapper.props(
      buildComponentContext("grafana.getAlertRule", {
        configuration: {
          alertRule: "rule-123",
        },
      }),
    );

    expect(props.metadata).toEqual([expect.objectContaining({ label: "rule-123" })]);
  });

  it("updateAlertRuleMapper uses configuration.alertRule when title is missing", () => {
    const props = updateAlertRuleMapper.props(
      buildComponentContext("grafana.updateAlertRule", {
        configuration: {
          alertRule: "rule-456",
          isPaused: true,
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ label: "rule-456" }),
        expect.objectContaining({ label: "Paused" }),
      ]),
    );
  });

  it("deleteAlertRuleMapper uses configuration.alertRule in metadata", () => {
    const props = deleteAlertRuleMapper.props(
      buildComponentContext("grafana.deleteAlertRule", {
        configuration: {
          alertRule: "rule-789",
        },
      }),
    );

    expect(props.metadata).toEqual([expect.objectContaining({ label: "rule-789" })]);
  });

  it("listAlertRulesMapper uses configuration.folder in metadata", () => {
    const props = listAlertRulesMapper.props(
      buildComponentContext("grafana.listAlertRules", {
        configuration: {
          folder: "folder-123",
          group: "service-health",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ label: "folder-123" }),
        expect.objectContaining({ label: "service-health" }),
      ]),
    );
  });

  it("alert rule mappers tolerate missing outputs", () => {
    const createCtx = buildExecutionContext("grafana.createAlertRule", { execution: { outputs: undefined } });
    const getCtx = buildExecutionContext("grafana.getAlertRule", { execution: { outputs: undefined } });
    const listCtx = buildExecutionContext("grafana.listAlertRules", { execution: { outputs: undefined } });
    const updateCtx = buildExecutionContext("grafana.updateAlertRule", { execution: { outputs: undefined } });
    const deleteCtx = buildExecutionContext("grafana.deleteAlertRule", { execution: { outputs: undefined } });

    expect(() => createAlertRuleMapper.getExecutionDetails(createCtx)).not.toThrow();
    expect(() => getAlertRuleMapper.getExecutionDetails(getCtx)).not.toThrow();
    expect(() => listAlertRulesMapper.getExecutionDetails(listCtx)).not.toThrow();
    expect(() => updateAlertRuleMapper.getExecutionDetails(updateCtx)).not.toThrow();
    expect(() => deleteAlertRuleMapper.getExecutionDetails(deleteCtx)).not.toThrow();
  });
});
