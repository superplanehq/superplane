import { describe, expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getSilenceMapper } from "./get_silence";
import type { PrometheusSilencePayload } from "./types";

describe("getSilenceMapper.getExecutionDetails", () => {
  it.each([
    ["undefined outputs", undefined],
    ["a missing default channel", {}],
    ["an empty default array", { default: [] }],
    ["missing data", { default: [{}] }],
    ["null data", { default: [{ data: null }] }],
  ] as const)("preserves Retrieved At with %s", (_name, outputs) => {
    const createdAt = new Date().toISOString();

    const details = getSilenceMapper.getExecutionDetails(buildDetailsContext({ createdAt, outputs }));

    expect(details).toEqual({ "Retrieved At": new Date(createdAt).toLocaleString() });
  });

  it("omits Retrieved At when the creation timestamp is absent", () => {
    const details = getSilenceMapper.getExecutionDetails(buildDetailsContext({ createdAt: "" }));

    expect(details).toEqual({});
  });

  it.each([{}, { matchers: [] }])("omits absent optional fields and empty matchers for %j", (silence) => {
    const details = getSilenceMapper.getExecutionDetails(
      buildDetailsContext({ createdAt: "", outputs: { default: [{ data: silence }] } }),
    );

    expect(details).toEqual({});
  });

  it("preserves populated silence details and matcher operators", () => {
    const createdAt = new Date().toISOString();
    const startsAt = new Date(Date.now() - 60_000).toISOString();
    const endsAt = new Date(Date.now() + 60_000).toISOString();
    const silence: PrometheusSilencePayload = {
      silenceID: "silence-1",
      status: "active",
      matchers: [
        { name: "alertname", value: "HighCPU", isEqual: true },
        { name: "environment", value: "development", isEqual: false },
        { name: "service", value: "api.*", isEqual: true, isRegex: true },
        { name: "instance", value: "test.*", isEqual: false, isRegex: true },
        { name: "team", value: "platform" },
      ],
      startsAt,
      endsAt,
      createdBy: "operator@example.com",
      comment: "Scheduled maintenance",
    };

    const details = getSilenceMapper.getExecutionDetails(
      buildDetailsContext({ createdAt, outputs: { default: [{ data: silence }] } }),
    );

    expect(details).toEqual({
      "Retrieved At": new Date(createdAt).toLocaleString(),
      "Silence ID": "silence-1",
      Status: "active",
      Matchers:
        'alertname="HighCPU", environment!="development", service="api.*" (regex), instance!="test.*" (regex), team="platform"',
      "Starts At": new Date(startsAt).toLocaleString(),
      "Ends At": new Date(endsAt).toLocaleString(),
      "Created By": "operator@example.com",
      Comment: "Scheduled maintenance",
    });
  });

  it("omits the sentinel end time", () => {
    const details = getSilenceMapper.getExecutionDetails(
      buildDetailsContext({
        createdAt: "",
        outputs: { default: [{ data: { silenceID: "silence-1", endsAt: "0001-01-01T00:00:00Z" } }] },
      }),
    );

    expect(details).toEqual({ "Silence ID": "silence-1" });
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Get Silence",
    componentName: "prometheus.getSilence",
    isCollapsed: false,
    configuration: {},
  };

  return {
    nodes: [node],
    node,
    execution: {
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
      ...execution,
    },
  };
}
