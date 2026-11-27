import { describe, expect, it, vi } from "bun:test";
import {
  NO_INCOMING_CONNECTIONS_WARNING,
  clearRunDetailNodeSearchParams,
  isValidRunId,
  isNotFoundError,
  prepareCanvasLogNodes,
  prepareSidebarData,
  shouldClearRunDetailNode,
  shouldClearStaleRunUrl,
  shouldRequestInitialRunFit,
  withDerivedNodeWarnings,
} from "./workflowPageHelpers";
import { makeComponentsNode, makeEdge } from "@/test/factories";
import type {
  ActionsAction,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  TriggersTrigger,
} from "@/api-client";
import { mapCanvasNodesToLogEntries } from "./utils";

const validRunId = "550e8400-e29b-41d4-a716-446655440000";

describe("prepareSidebarData", () => {
  for (const type of ["TYPE_ACTION", "TYPE_TRIGGER"] as const) {
    const definition: ActionsAction & TriggersTrigger = {
      name: "sidebar.component",
      label: "Definition label",
      icon: "activity",
      color: "green",
    };
    const conflictingDefinition: ActionsAction & TriggersTrigger = {
      ...definition,
      label: "Wrong definition",
      icon: "bell",
      color: "red",
    };
    const cases: {
      description: string;
      metadata?: ActionsAction & TriggersTrigger;
      nodeName?: string;
      title: string;
      iconSlug: string;
      iconColor: string;
    }[] = [
      {
        description: "uses the first matching definition instead of the node name",
        metadata: definition,
        nodeName: "Node name",
        title: "Definition label",
        iconSlug: "activity",
        iconColor: "text-green-600 dark:text-green-400",
      },
      {
        description: "falls back from an empty label to the node name",
        metadata: { ...definition, label: "" },
        nodeName: "Node name",
        title: "Node name",
        iconSlug: "activity",
        iconColor: "text-green-600 dark:text-green-400",
      },
      {
        description: "defaults empty icon and color values",
        metadata: { ...definition, icon: "", color: "" },
        nodeName: "Node name",
        title: "Definition label",
        iconSlug: "boxes",
        iconColor: "text-indigo-600 dark:text-indigo-400",
      },
      {
        description: "defaults missing metadata fields",
        metadata: { name: definition.name },
        nodeName: "Node name",
        title: "Node name",
        iconSlug: "boxes",
        iconColor: "text-indigo-600 dark:text-indigo-400",
      },
      {
        description: "uses the node name when only the opposite catalog matches",
        nodeName: "Node name",
        title: "Node name",
        iconSlug: "boxes",
        iconColor: "text-indigo-600 dark:text-indigo-400",
      },
      {
        description: "defaults missing definitions and empty node names",
        nodeName: "",
        title: "Unknown",
        iconSlug: "boxes",
        iconColor: "text-indigo-600 dark:text-indigo-400",
      },
      {
        description: "defaults empty labels and missing node names",
        metadata: { name: definition.name, label: "" },
        title: "Unknown",
        iconSlug: "boxes",
        iconColor: "text-indigo-600 dark:text-indigo-400",
      },
    ];

    it.each(cases)(`${type}: $description`, ({ metadata, nodeName, title, iconSlug, iconColor }) => {
      const node = makeComponentsNode({ type, component: definition.name, name: nodeName });
      const matchingCatalog = [
        { ...conflictingDefinition, name: "unrelated.component" },
        ...(metadata ? [metadata] : []),
        ...(metadata ? [conflictingDefinition] : []),
      ];
      const sidebar = prepareSidebarData({
        node,
        nodes: [node],
        components: type === "TYPE_ACTION" ? matchingCatalog : [conflictingDefinition],
        triggers: type === "TYPE_TRIGGER" ? matchingCatalog : [conflictingDefinition],
        nodeExecutionsMap: {},
        nodeQueueItemsMap: {},
        nodeEventsMap: {},
      });

      expect(sidebar).toMatchObject({
        title,
        iconSlug,
        iconColor,
        latestEvents: [],
        nextInQueueEvents: [],
        totalInHistoryCount: 0,
        totalInQueueCount: 0,
        hideQueueEvents: type === "TYPE_TRIGGER",
        isComposite: false,
      });
    });
  }

  it.each(["TYPE_ACTION", "TYPE_TRIGGER"] as const)(
    "%s: assembles limited history and queue with supplied counts",
    (type) => {
      const node = makeComponentsNode({ type });
      const createdAt = new Date().toISOString();
      const events: CanvasesCanvasEvent[] = Array.from({ length: 6 }, (_, index) => ({
        id: `event-${index}`,
        nodeId: node.id,
        createdAt,
      }));
      const executions: CanvasesCanvasNodeExecution[] = Array.from({ length: 6 }, (_, index) => ({
        id: `execution-${index}`,
        nodeId: node.id,
        createdAt,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
      }));
      const queueItems: CanvasesCanvasNodeQueueItem[] = Array.from({ length: 6 }, (_, index) => ({
        id: `queue-${index}`,
        createdAt,
      }));
      const sidebar = prepareSidebarData({
        node,
        nodes: [node],
        components: [],
        triggers: [],
        nodeExecutionsMap: { [node.id!]: executions },
        nodeQueueItemsMap: { [node.id!]: queueItems },
        nodeEventsMap: { [node.id!]: events },
        totalHistoryCount: 23,
        totalQueueCount: 12,
      });

      const historyPrefix = type === "TYPE_TRIGGER" ? "event" : "execution";
      expect(sidebar.latestEvents.map((event) => event.id)).toEqual(
        Array.from({ length: 5 }, (_, index) => `${historyPrefix}-${index}`),
      );
      expect(
        sidebar.latestEvents.every((event) => event.kind === (type === "TYPE_TRIGGER" ? "trigger" : "execution")),
      ).toBe(true);
      expect(
        sidebar.nextInQueueEvents.map((event) => ({ id: event.id, kind: event.kind, state: event.state })),
      ).toEqual(Array.from({ length: 5 }, (_, index) => ({ id: `queue-${index}`, kind: "queue", state: "queued" })));
      expect(sidebar).toMatchObject({
        totalInHistoryCount: 23,
        totalInQueueCount: 12,
        hideQueueEvents: type === "TYPE_TRIGGER",
        isComposite: false,
      });
    },
  );
});

describe("isNotFoundError", () => {
  it.each([
    "Not Found",
    "NOT FOUND",
    new Error("Not Found"),
    { status: 404 },
    { response: { status: 404 } },
    { code: "NOT_FOUND" },
    new Error("Request failed: 404"),
  ])("recognizes missing resources: %p", (error) => {
    expect(isNotFoundError(error)).toBe(true);
  });

  it.each([undefined, null, "Forbidden", new Error("Internal Server Error"), { status: 500 }])(
    "does not classify other failures as missing resources: %p",
    (error) => {
      expect(isNotFoundError(error)).toBe(false);
    },
  );
});

describe("workflowPageHelpers run inspection", () => {
  it("clears stale run URLs after describe settles without a run", () => {
    expect(
      shouldClearStaleRunUrl({
        selectedRunId: validRunId,
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: true,
        describeRunSettled: false,
      }),
    ).toBe(false);

    expect(
      shouldClearStaleRunUrl({
        selectedRunId: validRunId,
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: false,
        describeRunSettled: true,
      }),
    ).toBe(true);
  });

  it("clears malformed run ids immediately", () => {
    expect(isValidRunId("not-a-uuid")).toBe(false);
    expect(
      shouldClearStaleRunUrl({
        selectedRunId: "not-a-uuid",
        isRunInspectionMode: true,
        selectedRun: null,
        isRunResolveLoading: false,
        describeRunSettled: false,
      }),
    ).toBe(true);
  });

  it("does not clear when the run resolved", () => {
    expect(
      shouldClearStaleRunUrl({
        selectedRunId: validRunId,
        isRunInspectionMode: true,
        selectedRun: { id: validRunId },
        isRunResolveLoading: false,
        describeRunSettled: true,
      }),
    ).toBe(false);
  });

  it("clears run detail nodes that are not part of the selected run", () => {
    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-b",
        participantNodeIds: ["node-a"],
        runCanvasLoading: false,
        runCanvasSettled: true,
      }),
    ).toBe(true);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: ["node-a"],
        runCanvasLoading: false,
        runCanvasSettled: true,
      }),
    ).toBe(false);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: false,
        runCanvasSettled: true,
      }),
    ).toBe(true);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: true,
        runCanvasSettled: false,
      }),
    ).toBe(false);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: false,
        runCanvasSettled: false,
      }),
    ).toBe(false);

    expect(
      shouldClearRunDetailNode({
        runDetailNodeId: "node-a",
        participantNodeIds: [],
        runCanvasLoading: false,
        runCanvasSettled: true,
      }),
    ).toBe(true);
  });

  it("clears matching stale run detail node search params", () => {
    const cleared = clearRunDetailNodeSearchParams(
      new URLSearchParams({ run: "run-1", sidebar: "1", node: "node-a" }),
      "node-a",
    );

    expect(cleared.get("run")).toBe("run-1");
    expect(cleared.get("sidebar")).toBeNull();
    expect(cleared.get("node")).toBeNull();
  });

  it("keeps run detail node search params for a newer URL selection", () => {
    const unchanged = clearRunDetailNodeSearchParams(
      new URLSearchParams({ run: "run-1", sidebar: "1", node: "node-b" }),
      "node-a",
    );

    expect(unchanged.get("sidebar")).toBe("1");
    expect(unchanged.get("node")).toBe("node-b");
  });
});

describe("shouldRequestInitialRunFit", () => {
  it("requests a fit when entering run inspection with a run and no pending node focus", () => {
    expect(
      shouldRequestInitialRunFit({
        isRunInspectionMode: true,
        selectedRunId: validRunId,
        searchParams: new URLSearchParams({ run: validRunId }),
      }),
    ).toBe(true);
  });

  it("does not request a fit outside of run inspection mode", () => {
    expect(
      shouldRequestInitialRunFit({
        isRunInspectionMode: false,
        selectedRunId: validRunId,
        searchParams: new URLSearchParams({ run: validRunId }),
      }),
    ).toBe(false);
  });

  it("does not request a fit without a selected run", () => {
    expect(
      shouldRequestInitialRunFit({
        isRunInspectionMode: true,
        selectedRunId: null,
        searchParams: new URLSearchParams(),
      }),
    ).toBe(false);
  });

  it("does not request a fit when a node is already pending focus via the sidebar", () => {
    expect(
      shouldRequestInitialRunFit({
        isRunInspectionMode: true,
        selectedRunId: validRunId,
        searchParams: new URLSearchParams({ run: validRunId, sidebar: "1", node: "node-a" }),
      }),
    ).toBe(false);
  });

  it("still requests a fit when sidebar param is set without a node", () => {
    expect(
      shouldRequestInitialRunFit({
        isRunInspectionMode: true,
        selectedRunId: validRunId,
        searchParams: new URLSearchParams({ run: validRunId, sidebar: "1" }),
      }),
    ).toBe(true);
  });
});

describe("withDerivedNodeWarnings", () => {
  const componentDefinitions = [
    {
      name: "source",
      outputChannels: [{ name: "success" }],
    },
    {
      name: "target",
      outputChannels: [{ name: "default" }],
    },
  ] as ActionsAction[];

  it("warns action nodes without valid incoming connections", () => {
    const source = makeComponentsNode({ id: "source", component: "source" });
    const target = makeComponentsNode({ id: "target", component: "target" });

    const [preparedSource, preparedTarget] = withDerivedNodeWarnings(
      [source, target],
      [makeEdge({ sourceId: "source", targetId: "target", channel: "default" })],
      componentDefinitions,
    );

    expect(preparedSource.warningMessage).toBe(NO_INCOMING_CONNECTIONS_WARNING);
    expect(preparedTarget.warningMessage).toBe(NO_INCOMING_CONNECTIONS_WARNING);
  });

  it("does not warn action nodes with valid incoming connections", () => {
    const source = makeComponentsNode({ id: "source", component: "source" });
    const target = makeComponentsNode({ id: "target", component: "target" });

    const prepared = withDerivedNodeWarnings(
      [source, target],
      [makeEdge({ sourceId: "source", targetId: "target", channel: "success" })],
      componentDefinitions,
    );

    expect(prepared.find((node) => node.id === "target")?.warningMessage).toBeUndefined();
  });

  it("does not warn trigger nodes or replace existing warnings", () => {
    const trigger = makeComponentsNode({ id: "trigger", type: "TYPE_TRIGGER" });
    const target = makeComponentsNode({ id: "target", component: "target", warningMessage: "Existing warning" });

    const prepared = withDerivedNodeWarnings([trigger, target], [], componentDefinitions);

    expect(prepared.find((node) => node.id === "trigger")?.warningMessage).toBeUndefined();
    expect(prepared.find((node) => node.id === "target")?.warningMessage).toBe("Existing warning");
  });
});

describe("prepareCanvasLogNodes", () => {
  it("includes derived connectivity warnings in canvas log entries", () => {
    const source = makeComponentsNode({ id: "source", name: "Source", component: "source" });
    const target = makeComponentsNode({ id: "target", name: "Target", component: "target" });
    const logNodes = prepareCanvasLogNodes(
      [source, target],
      [makeEdge({ sourceId: "source", targetId: "target", channel: "default" })],
      [
        {
          name: "source",
          outputChannels: [{ name: "success" }],
        },
        {
          name: "target",
          outputChannels: [{ name: "default" }],
        },
      ] as ActionsAction[],
      true,
    );

    const entries = mapCanvasNodesToLogEntries({
      nodes: logNodes,
      workflowUpdatedAt: "2026-07-06T16:00:00Z",
      onNodeSelect: vi.fn(),
    });

    expect(entries).toHaveLength(2);
    expect(entries.map((entry) => entry.searchText)).toEqual([
      `Source source ${NO_INCOMING_CONNECTIONS_WARNING}`,
      `Target target ${NO_INCOMING_CONNECTIONS_WARNING}`,
    ]);
  });

  it("keeps log nodes unchanged until component metadata is ready", () => {
    const source = makeComponentsNode({ id: "source", name: "Source", component: "source" });
    const target = makeComponentsNode({ id: "target", name: "Target", component: "target" });

    const logNodes = prepareCanvasLogNodes(
      [source, target],
      [makeEdge({ sourceId: "source", targetId: "target", channel: "default" })],
      [] as ActionsAction[],
      false,
    );

    expect(logNodes).toEqual([source, target]);
  });
});
