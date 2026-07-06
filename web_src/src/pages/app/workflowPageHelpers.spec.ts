import { describe, expect, it } from "vitest";
import {
  NO_INCOMING_CONNECTIONS_WARNING,
  clearRunDetailNodeSearchParams,
  isValidRunId,
  shouldClearRunDetailNode,
  shouldClearStaleRunUrl,
  withDerivedNodeWarnings,
} from "./workflowPageHelpers";
import { makeComponentsNode, makeEdge } from "@/test/factories";
import type { ActionsAction } from "@/api-client";

const validRunId = "550e8400-e29b-41d4-a716-446655440000";

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
