import { describe, expect, it } from "vitest";
import type { ActionsAction, CanvasesCanvasNodeExecution, SuperplaneComponentsNode, TriggersTrigger } from "@/api-client";
import { evaluateExpr } from "@/lib/exprEvaluator";
import { getSuggestions } from "@/components/AutoCompleteInput/core";
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

  // Regression test for issue #5944: expression autocomplete broke after `.data`
  // on runner node output. The runner emits a single object under `data`
  // ({type, timestamp, data: {status, exit_code, result}}), so autocomplete must
  // keep suggesting the object's fields past `.data` — array indexing there
  // (`.data[0]`) resolves to nothing at runtime.
  describe("runner node output (issue #5944)", () => {
    const runnerNode: SuperplaneComponentsNode = {
      id: "runner-1",
      name: "Fetch GitHub Stats",
      type: "TYPE_ACTION",
      component: "runner.runPython",
    };

    const discordNode: SuperplaneComponentsNode = {
      id: "discord-1",
      name: "Discord",
      type: "TYPE_ACTION",
      component: "discord.sendMessage",
    };

    const runnerData = {
      status: "succeeded",
      exit_code: 0,
      result: { closed_last_7_days: { count: 3 } },
    };

    const runnerMetadata: ActionsAction = {
      name: "runner.runPython",
      label: "Run Python",
      exampleOutput: {
        type: "runner.finished",
        timestamp: "2026-01-16T17:56:16.680755501Z",
        data: runnerData,
      },
    };

    function buildDiscordContext(overrides: Partial<AutocompleteExampleContext> = {}): AutocompleteExampleContext {
      return makeContext({
        canvasNodes: [runnerNode, discordNode],
        canvasNodesById: new Map([
          [runnerNode.id!, runnerNode],
          [discordNode.id!, discordNode],
        ]),
        incomingNodeIdsByTargetId: new Map([[discordNode.id!, [runnerNode.id!]]]),
        allComponentsByName: new Map([[runnerMetadata.name, runnerMetadata]]),
        ...overrides,
      });
    }

    function fieldLabelsAfter(exampleObj: Record<string, unknown>, expression: string): string[] {
      return getSuggestions(expression, expression.length, exampleObj, {
        includeFunctions: false,
      }).map((suggestion) => suggestion.label);
    }

    it("keeps autocompleting object fields past `.data` using example output", () => {
      const autocompleteContext = buildAutocompleteExampleObj(discordNode.id!, buildDiscordContext());
      expect(autocompleteContext).not.toBeNull();

      expect(fieldLabelsAfter(autocompleteContext!, '$["Fetch GitHub Stats"].data.')).toEqual(
        expect.arrayContaining(["status", "exit_code", "result"]),
      );
      expect(fieldLabelsAfter(autocompleteContext!, '$["Fetch GitHub Stats"].data.result.')).toContain(
        "closed_last_7_days",
      );
      expect(fieldLabelsAfter(autocompleteContext!, '$["Fetch GitHub Stats"].data.result.closed_last_7_days.')).toContain(
        "count",
      );
      expect(
        evaluateExpr('$["Fetch GitHub Stats"].data.result.closed_last_7_days.count', autocompleteContext!),
      ).toBe(3);
    });

    it("keeps autocompleting object fields past `.data` from the latest execution output", () => {
      const execution: CanvasesCanvasNodeExecution = {
        state: "STATE_FINISHED",
        resultReason: "RESULT_REASON_OK",
        outputs: {
          passed: [
            {
              type: "runner.finished",
              timestamp: "2026-01-16T17:56:16.680755501Z",
              data: runnerData,
            },
          ],
        },
      } as CanvasesCanvasNodeExecution;

      const autocompleteContext = buildAutocompleteExampleObj(
        discordNode.id!,
        buildDiscordContext({ visibleNodeExecutionsMap: { [runnerNode.id!]: [execution] } }),
      );
      expect(autocompleteContext).not.toBeNull();

      expect(fieldLabelsAfter(autocompleteContext!, '$["Fetch GitHub Stats"].data.')).toEqual(
        expect.arrayContaining(["status", "exit_code", "result"]),
      );
      expect(
        evaluateExpr('$["Fetch GitHub Stats"].data.result.closed_last_7_days.count', autocompleteContext!),
      ).toBe(3);
    });
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
