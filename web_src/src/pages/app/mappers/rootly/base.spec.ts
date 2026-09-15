import { describe, expect, it } from "bun:test";
import { baseEventSections } from "./base";
import { defaultTriggerRenderer } from "../default";
import type { ExecutionInfo } from "../types";

function makeExecution(): ExecutionInfo {
  const createdAt = new Date().toISOString();
  return {
    id: "exec-1",
    createdAt,
    updatedAt: createdAt,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: {
      id: "event-1",
      createdAt,
      data: {},
      nodeId: "missing-trigger",
      type: "rootly.onIncident",
    },
  };
}

describe("rootly baseEventSections", () => {
  it("uses the default trigger renderer when the root trigger node is absent", () => {
    const execution = makeExecution();
    const sections = baseEventSections([], execution, "rootly.getIncident");
    const expectedTitle = defaultTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent }).title;

    expect(sections).toHaveLength(1);
    expect(sections[0].eventTitle).toBe(expectedTitle);
  });
});
