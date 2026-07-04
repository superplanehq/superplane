import { describe, expect, it } from "vitest";
import type { SuperplaneComponentsNode, TriggersTrigger } from "@/api-client";
import { evaluateExpr } from "@/lib/exprEvaluator";
import { buildAutocompleteExampleObj, type AutocompleteExampleContext } from "./buildAutocompleteExampleObj";

const triggerNode: SuperplaneComponentsNode = {
  id: "trigger-1",
  name: "GitHub Check Run",
  type: "TYPE_TRIGGER",
  component: "github.onCheckRun",
};

const triggerMetadata: TriggersTrigger = {
  name: "github.onCheckRun",
  label: "Check Run",
  exampleData: {
    type: "github.checkRun",
    timestamp: "2026-06-12T08:00:00Z",
    data: {
      check_run: {
        name: "DCO",
        conclusion: "success",
        head_sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
      },
    },
  },
};

function makeContext(overrides: Partial<AutocompleteExampleContext>): AutocompleteExampleContext {
  return {
    canvasNodes: [],
    canvasNodesById: new Map(),
    incomingNodeIdsByTargetId: new Map(),
    visibleNodeExecutionsMap: {},
    visibleNodeEventsMap: {},
    allComponentsByName: new Map(),
    allTriggersByName: new Map(),
    ...overrides,
  };
}

describe("buildAutocompleteExampleObj", () => {
  it("keeps root context when editing a trigger without upstream nodes", () => {
    const autocompleteContext = buildAutocompleteExampleObj(
      triggerNode.id!,
      makeContext({
        canvasNodes: [triggerNode],
        canvasNodesById: new Map([[triggerNode.id!, triggerNode]]),
        allTriggersByName: new Map([[triggerMetadata.name, triggerMetadata]]),
      }),
    );

    expect(autocompleteContext).toEqual({
      __root: triggerMetadata.exampleData,
    });
    expect(
      evaluateExpr(
        "root().data.check_run.name + ' ' + root().data.check_run.conclusion + ' - ' + root().data.check_run.head_sha[:7]",
        autocompleteContext!,
      ),
    ).toBe("DCO success - d6f3c8a");
  });

  it("uses the latest trigger event before falling back to example data", () => {
    const latestEventData = {
      type: "github.checkRun",
      timestamp: "2026-06-12T09:00:00Z",
      data: {
        check_run: {
          name: "Unit tests",
          conclusion: "failure",
          head_sha: "abcdef1234567890",
        },
      },
    };

    const autocompleteContext = buildAutocompleteExampleObj(
      triggerNode.id!,
      makeContext({
        canvasNodes: [triggerNode],
        canvasNodesById: new Map([[triggerNode.id!, triggerNode]]),
        visibleNodeEventsMap: {
          [triggerNode.id!]: [{ data: latestEventData }],
        },
        allTriggersByName: new Map([[triggerMetadata.name, triggerMetadata]]),
      }),
    );

    expect(autocompleteContext).toEqual({
      __root: latestEventData,
    });
    expect(evaluateExpr("root().data.check_run.name", autocompleteContext!)).toBe("Unit tests");
  });
});
