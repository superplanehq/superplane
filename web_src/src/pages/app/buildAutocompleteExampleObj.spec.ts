import { describe, expect, it } from "vitest";
import type {
  ActionsAction,
  CanvasesCanvasNodeExecution,
  SuperplaneComponentsNode,
  TriggersTrigger,
} from "@/api-client";
import { evaluateExpr } from "@/lib/exprEvaluator";
import {
  buildAutocompleteExampleObj,
  buildAutocompleteExampleResult,
  summarizePayloadSources,
  type AutocompleteExampleContext,
} from "./buildAutocompleteExampleObj";

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
    nodeExecutionsMap: {},
    nodeEventsMap: {},
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
      __app: expect.objectContaining({
        id: expect.any(String),
        name: "Example App",
        description: "",
        url: expect.any(String),
      }),
      __run: expect.objectContaining({
        id: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        url: expect.stringContaining("?run=f47ac10b-58cc-4372-a567-0e02b2c3d479"),
        started_at: expect.any(String),
      }),
      __order: expect.objectContaining({
        id: expect.any(String),
        title: "Ship feature",
        artifacts: expect.any(Array),
        comments: expect.any(Array),
      }),
      __workspace: expect.objectContaining({
        id: expect.any(String),
        name: "Example workspace",
        repository: "acme/service",
        default_branch: "main",
      }),
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
        nodeEventsMap: {
          [triggerNode.id!]: [{ id: "evt-1", data: latestEventData, createdAt: "2026-06-12T09:00:00Z" }],
        },
        allTriggersByName: new Map([[triggerMetadata.name, triggerMetadata]]),
        app: {
          id: "canvas-1",
          name: "Deploy",
          description: "Production deploy",
        },
      }),
    );

    expect(autocompleteContext).toEqual({
      __root: latestEventData,
      __app: expect.objectContaining({
        id: "canvas-1",
        name: "Deploy",
        description: "Production deploy",
        url: expect.any(String),
      }),
      __run: expect.objectContaining({
        id: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        url: expect.stringContaining("?run=f47ac10b-58cc-4372-a567-0e02b2c3d479"),
        started_at: expect.any(String),
      }),
      __order: expect.objectContaining({
        id: expect.any(String),
        title: "Ship feature",
        artifacts: expect.any(Array),
        comments: expect.any(Array),
      }),
      __workspace: expect.objectContaining({
        id: expect.any(String),
        name: "Example workspace",
        repository: "acme/service",
        default_branch: "main",
      }),
    });
    expect(evaluateExpr("root().data.check_run.name", autocompleteContext!)).toBe("Unit tests");
    expect(evaluateExpr("app().name", autocompleteContext!)).toBe("Deploy");
    expect(evaluateExpr("order().title", autocompleteContext!)).toBe("Ship feature");
    expect(evaluateExpr("task().title", autocompleteContext!)).toBe("Ship feature");
  });
});

// A two-node workflow mirroring issue #6404: Schedule trigger -> HTTP component
// (`read-funnel`) -> If component (the node being authored).
const readFunnelNode: SuperplaneComponentsNode = {
  id: "node-read-funnel",
  name: "read-funnel",
  type: "TYPE_ACTION",
  component: "http.request",
};

const ifNode: SuperplaneComponentsNode = {
  id: "node-if",
  name: "check-accepted",
  type: "TYPE_ACTION",
  component: "control.if",
};

const httpAction: ActionsAction = {
  name: "http.request",
  label: "HTTP Request",
  exampleOutput: { message: "ok" },
};

function finishedExecution(overrides: Partial<CanvasesCanvasNodeExecution> = {}): CanvasesCanvasNodeExecution {
  return {
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    ...overrides,
  };
}

function makeChainContext(overrides: Partial<AutocompleteExampleContext>): AutocompleteExampleContext {
  return makeContext({
    canvasNodes: [readFunnelNode, ifNode],
    canvasNodesById: new Map([
      [readFunnelNode.id!, readFunnelNode],
      [ifNode.id!, ifNode],
    ]),
    // If component is being authored; read-funnel is its upstream.
    incomingNodeIdsByTargetId: new Map([[ifNode.id!, [readFunnelNode.id!]]]),
    allComponentsByName: new Map([[httpAction.name, httpAction]]),
    ...overrides,
  });
}

describe("buildAutocompleteExampleResult - action node payloads", () => {
  it("uses a real action output so the editor evaluates the issue's expression as true", () => {
    const realOutput = { data: { body: { jobs: { accepted: 9 } } } };
    const context = makeChainContext({
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({
            id: "exec-real",
            nodeId: readFunnelNode.id!,
            outputs: { default: [realOutput] },
            createdAt: "2026-07-31T00:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    // The embedded example would be `{ message: "ok" }`; the real payload must win.
    expect(evaluateExpr('$["read-funnel"].data.body.jobs.accepted > 0', result.context!)).toBe(true);
    expect(summarizePayloadSources(result.sourcesByNodeId)?.label).toBe("Latest real payload");
  });

  it("prefers real trigger event data over the trigger's embedded exampleData", () => {
    const triggerChainNode: SuperplaneComponentsNode = {
      id: "trigger-sched",
      name: "every-minute",
      type: "TYPE_TRIGGER",
      component: "schedule.interval",
    };
    const triggerMeta: TriggersTrigger = {
      name: "schedule.interval",
      label: "Schedule",
      exampleData: { data: { fired: "example" } },
    };
    const ctx = makeContext({
      canvasNodes: [triggerChainNode, ifNode],
      canvasNodesById: new Map([
        [triggerChainNode.id!, triggerChainNode],
        [ifNode.id!, ifNode],
      ]),
      incomingNodeIdsByTargetId: new Map([[ifNode.id!, [triggerChainNode.id!]]]),
      allTriggersByName: new Map([[triggerMeta.name, triggerMeta]]),
      nodeEventsMap: {
        [triggerChainNode.id!]: [
          { id: "evt-real", data: { data: { fired: "2026-07-31T00:00:00Z" } }, createdAt: "2026-07-31T00:00:00Z" },
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, ctx);

    expect(evaluateExpr('$["every-minute"].data.fired', result.context!)).toBe("2026-07-31T00:00:00Z");
    expect(summarizePayloadSources(result.sourcesByNodeId)?.label).toBe("Latest trigger event");
  });

  it("skips a running execution and uses the newest finished execution with usable output", () => {
    const olderOutput = { data: { body: { jobs: { accepted: 4 } } } };
    const newerOutput = { data: { body: { jobs: { accepted: 12 } } } };
    const context = makeChainContext({
      // Store/API order is newest-first (created_at DESC). A running execution
      // must not hide the newest finished execution with usable output.
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({ id: "exec-running", state: "STATE_STARTED", outputs: {} }),
          finishedExecution({
            id: "exec-new",
            outputs: { default: [newerOutput] },
            createdAt: "2026-07-31T02:00:00Z",
          }),
          finishedExecution({
            id: "exec-old",
            outputs: { default: [olderOutput] },
            createdAt: "2026-07-31T01:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    expect(evaluateExpr('$["read-funnel"].data.body.jobs.accepted', result.context!)).toBe(12);
    expect(result.sourcesByNodeId[readFunnelNode.id!]).toEqual({
      kind: "execution",
      executionId: "exec-new",
      observedAt: "2026-07-31T02:00:00Z",
    });
  });

  it("skips a finished-but-errored execution and keeps the most recent usable real payload", () => {
    const usableOutput = { data: { body: { jobs: { accepted: 7 } } } };
    const context = makeChainContext({
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({
            id: "exec-failed",
            resultReason: "RESULT_REASON_ERROR",
            outputs: { default: [{ data: { body: { jobs: { accepted: 0 } } } }] },
            createdAt: "2026-07-31T03:00:00Z",
          }),
          finishedExecution({
            id: "exec-ok",
            outputs: { default: [usableOutput] },
            createdAt: "2026-07-31T02:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    expect(evaluateExpr('$["read-funnel"].data.body.jobs.accepted', result.context!)).toBe(7);
    expect(result.sourcesByNodeId[readFunnelNode.id!]).toEqual(
      expect.objectContaining({ kind: "execution", executionId: "exec-ok" }),
    );
  });

  it("uses a finished cancelled execution's real output but never labels it successful", () => {
    // A finished execution whose result is CANCELLED (with a non-error reason)
    // still carries a usable payload. The selector keeps the output; the label
    // stays "Latest real payload", not "successful".
    const cancelledOutput = { data: { body: { jobs: { accepted: 2 } } } };
    const context = makeChainContext({
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({
            id: "exec-cancelled",
            result: "RESULT_CANCELLED",
            resultReason: "RESULT_REASON_OK",
            outputs: { default: [cancelledOutput] },
            createdAt: "2026-07-31T00:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    expect(evaluateExpr('$["read-funnel"].data.body.jobs.accepted', result.context!)).toBe(2);
    expect(result.sourcesByNodeId[readFunnelNode.id!]).toEqual(
      expect.objectContaining({ kind: "execution", executionId: "exec-cancelled" }),
    );
    expect(summarizePayloadSources(result.sourcesByNodeId)?.label).toBe("Latest real payload");
  });

  it("uses a resolved-error execution's real output but never labels it successful", () => {
    // RESULT_REASON_ERROR_RESOLVED is not filtered (only RESULT_REASON_ERROR
    // is). The output stays usable; the label is "Latest real payload".
    const resolvedOutput = { data: { body: { jobs: { accepted: 6 } } } };
    const context = makeChainContext({
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({
            id: "exec-resolved",
            result: "RESULT_FAILED",
            resultReason: "RESULT_REASON_ERROR_RESOLVED",
            outputs: { default: [resolvedOutput] },
            createdAt: "2026-07-31T00:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    expect(evaluateExpr('$["read-funnel"].data.body.jobs.accepted', result.context!)).toBe(6);
    expect(summarizePayloadSources(result.sourcesByNodeId)?.label).toBe("Latest real payload");
  });

  it("falls back to the embedded example output when no usable real execution exists", () => {
    const context = makeChainContext({
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({ id: "exec-empty", outputs: { default: [] }, createdAt: "2026-07-31T01:00:00Z" }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    expect(evaluateExpr('$["read-funnel"].message', result.context!)).toBe("ok");
    expect(result.sourcesByNodeId[readFunnelNode.id!]).toEqual({ kind: "example" });
    expect(summarizePayloadSources(result.sourcesByNodeId)?.label).toBe("Example payload");
  });

  it("injects configuration from the same execution that supplied the payload", () => {
    const usableOutput = { data: { body: { jobs: { accepted: 3 } } } };
    const chosenConfig = { method: "GET", url: "https://example.com/funnel" };
    const context = makeChainContext({
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({
            id: "exec-with-config",
            outputs: { default: [usableOutput] },
            configuration: chosenConfig,
            createdAt: "2026-07-31T02:00:00Z",
          }),
          // An older finished execution with a different config does not win.
          finishedExecution({
            id: "exec-other-config",
            outputs: { default: [{ data: { body: { jobs: { accepted: 1 } } } }] },
            configuration: { method: "POST" },
            createdAt: "2026-07-31T01:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    expect(evaluateExpr('$["read-funnel"].config.url', result.context!)).toBe("https://example.com/funnel");
    expect(result.sourcesByNodeId[readFunnelNode.id!]).toEqual(
      expect.objectContaining({ kind: "execution", executionId: "exec-with-config" }),
    );
  });

  it("keeps provenance out of the evaluated payload (no fake keys)", () => {
    const context = makeChainContext({
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({
            id: "exec-clean",
            outputs: { default: [{ data: { body: { jobs: { accepted: 9 } } } }] },
            createdAt: "2026-07-31T00:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);
    const payload = (result.context!["read-funnel"] as Record<string, unknown>) ?? {};

    // Provenance must live only in sourcesByNodeId, never as payload properties.
    expect("__payloadSource" in payload).toBe(false);
    expect("__source" in payload).toBe(false);
    expect("__isExample" in payload).toBe(false);
    expect(evaluateExpr('$["read-funnel"].__payloadSource', result.context!)).toBeUndefined();
  });

  it("uses an older execution's real output when a newer finished execution has no output", () => {
    const olderOutput = { data: { body: { jobs: { accepted: 5 } } } };
    const context = makeChainContext({
      nodeExecutionsMap: {
        // Newest first: the newest finished execution carries no usable output,
        // so it must not mask the older execution that does have a real payload.
        [readFunnelNode.id!]: [
          finishedExecution({ id: "exec-empty", outputs: { default: [] }, createdAt: "2026-07-31T03:00:00Z" }),
          finishedExecution({
            id: "exec-real",
            outputs: { default: [olderOutput] },
            createdAt: "2026-07-31T02:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    expect(evaluateExpr('$["read-funnel"].data.body.jobs.accepted', result.context!)).toBe(5);
    expect(result.sourcesByNodeId[readFunnelNode.id!]).toEqual(
      expect.objectContaining({ kind: "execution", executionId: "exec-real" }),
    );
  });

  it("preserves array payloads instead of converting them to numeric-keyed objects", () => {
    const context = makeChainContext({
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({
            id: "exec-array",
            outputs: { default: [["alpha", "beta", "gamma"]] },
            createdAt: "2026-07-31T00:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    expect(Array.isArray(result.context!["read-funnel"])).toBe(true);
    expect(evaluateExpr('$["read-funnel"][0]', result.context!)).toBe("alpha");
    expect(evaluateExpr('$["read-funnel"][2]', result.context!)).toBe("gamma");
  });

  it("on example fallback injects the node's own configuration, not an unrelated execution's", () => {
    // A real execution exists but has no usable output, so the payload falls
    // back to the embedded example. The node's own config is attached; the
    // output-less execution's config is not.
    const context = makeChainContext({
      canvasNodesById: new Map([
        [readFunnelNode.id!, { ...readFunnelNode, configuration: { url: "node-config-url" } }],
        [ifNode.id!, ifNode],
      ]),
      nodeExecutionsMap: {
        [readFunnelNode.id!]: [
          finishedExecution({
            id: "exec-no-output",
            outputs: { default: [] },
            configuration: { url: "execution-config-url" },
            createdAt: "2026-07-31T00:00:00Z",
          }),
        ],
      },
    });

    const result = buildAutocompleteExampleResult(ifNode.id!, context);

    expect(evaluateExpr('$["read-funnel"].message', result.context!)).toBe("ok");
    expect(evaluateExpr('$["read-funnel"].config.url', result.context!)).toBe("node-config-url");
    expect(result.sourcesByNodeId[readFunnelNode.id!]).toEqual({ kind: "example" });
  });
});

describe("summarizePayloadSources", () => {
  it("returns null when there are no contributing nodes", () => {
    expect(summarizePayloadSources({})).toBeNull();
  });

  it("labels a mixed real + example graph as 'Includes example data'", () => {
    const summary = summarizePayloadSources({
      a: { kind: "execution" },
      b: { kind: "event" },
      c: { kind: "example" },
    });
    // Some real data + some example data. Flagged as example-including so the
    // badge does not imply the whole preview is real.
    expect(summary?.label).toBe("Includes example data");
    expect(summary?.isExample).toBe(true);
  });

  it("reserves 'Example payload' for all-example contexts", () => {
    const summary = summarizePayloadSources({
      a: { kind: "example" },
      b: { kind: "example" },
    });
    expect(summary?.label).toBe("Example payload");
    expect(summary?.isExample).toBe(true);
  });

  it("labels mixed execution + event (no examples) as 'Latest real data'", () => {
    const summary = summarizePayloadSources({
      a: { kind: "execution" },
      b: { kind: "event" },
    });
    expect(summary?.label).toBe("Latest real data");
    expect(summary?.isExample).toBe(false);
  });
});
