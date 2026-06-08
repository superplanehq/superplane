import { describe, it, expect } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client/types.gen";

import { buildRowPayloadFromTemplates, mergeTriggerParameters } from "./mergeTriggerPayload";

const START_NODE: SuperplaneComponentsNode = {
  id: "n1",
  name: "start",
  type: "TYPE_TRIGGER",
  configuration: {
    templates: [{ name: "deploy", payload: { issue: { number: 0 } } }],
  },
};

describe("mergeTriggerPayload", () => {
  it("merges row payload templates into run hook parameters", () => {
    const row = { pr_number: "99" };
    const params = mergeTriggerParameters(START_NODE, "run", "deploy", row, {
      "issue.number": "{{ pr_number }}",
    });
    expect(params).toEqual({
      template: "deploy",
      issue: { number: "99" },
    });
  });

  it("merges row payload templates into custom hook parameters", () => {
    const row = { reason: "manual" };
    const params = mergeTriggerParameters(START_NODE, "approve", "deploy", row, {
      reason: "{{ reason }}",
    });
    expect(params).toEqual({
      reason: "manual",
    });
  });

  it("builds nested paths from templates", () => {
    const out = buildRowPayloadFromTemplates({ "data.issue.number": "{{ pr_number }}" }, { pr_number: "7" });
    expect(out).toEqual({ data: { issue: { number: "7" } } });
  });

  it("coerces numeric strings for arithmetic templates like `{{ value / 2 }}`", () => {
    const out = buildRowPayloadFromTemplates({ amount: "{{ value / 2 }}" }, { value: "10" });
    expect(out).toEqual({ amount: "5" });
  });

  it("falls back to an empty string when a CEL operand cannot be coerced", () => {
    const out = buildRowPayloadFromTemplates({ amount: "{{ value / 2 }}" }, { value: "abc" });
    // evalExpr returns undefined → evalTemplate skips that segment, leaving an
    // empty string (the literal prefix in this template is also empty).
    expect(out).toEqual({ amount: "" });
  });
});
