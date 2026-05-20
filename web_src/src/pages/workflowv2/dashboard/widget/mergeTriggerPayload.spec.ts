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
  it("merges row payload into template defaults", () => {
    const row = { pr_number: "99" };
    const params = mergeTriggerParameters(START_NODE, "run", "deploy", row, {
      "issue.number": "{{ pr_number }}",
    });
    expect(params).toEqual({
      template: "deploy",
      payload: { issue: { number: "99" } },
    });
  });

  it("builds nested paths from templates", () => {
    const out = buildRowPayloadFromTemplates({ "data.issue.number": "{{ pr_number }}" }, { pr_number: "7" });
    expect(out).toEqual({ data: { issue: { number: "7" } } });
  });
});
