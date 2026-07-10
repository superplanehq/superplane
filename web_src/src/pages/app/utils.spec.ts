import { describe, expect, it, vi } from "vitest";
import { makeComponentsNode } from "@/test/factories";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import {
  findRunIdForSidebarEvent,
  mapCanvasNodesToLogEntries,
  mapExecutionsToSidebarEvents,
  mapTriggerEventToSidebarEvent,
} from "./utils";

describe("mapCanvasNodesToLogEntries", () => {
  it("maps node warnings into canvas log entries", () => {
    const entries = mapCanvasNodesToLogEntries({
      nodes: [
        makeComponentsNode({
          id: "draft-node-newer",
          name: "Draft Node Newer",
          warningMessage: "Newer warning",
        }),
        makeComponentsNode({
          id: "draft-node-older",
          name: "Draft Node Older",
          warningMessage: "Older warning",
        }),
      ],
      workflowUpdatedAt: "2026-04-03T12:00:00Z",
      onNodeSelect: vi.fn(),
    });

    expect(entries).toHaveLength(2);
    expect(entries.map((entry) => entry.id)).toEqual(["warning-1", "warning-2"]);
    expect(entries.every((entry) => entry.type === "warning")).toBe(true);
    expect(entries.every((entry) => entry.source === "canvas")).toBe(true);
    expect(entries[1]?.searchText).toContain("Older warning");
  });
});

describe("findRunIdForSidebarEvent", () => {
  it("uses an explicit run id from the sidebar event before loaded runs", () => {
    const event = {
      id: "execution-1",
      title: "Schedule",
      state: "success",
      isOpen: false,
      kind: "execution",
      runId: "run-from-status",
      executionId: "execution-1",
    } satisfies SidebarEvent;

    expect(findRunIdForSidebarEvent([{ id: "loaded-run", executions: [{ id: "execution-1" }] }], event)).toBe(
      "run-from-status",
    );
  });

  it("matches a run by root event id from a sidebar execution event", () => {
    const event = {
      id: "execution-1",
      title: "Schedule",
      state: "success",
      isOpen: false,
      kind: "execution",
      triggerEventId: "root-event-1",
    } satisfies SidebarEvent;

    expect(
      findRunIdForSidebarEvent([{ id: "run-1", rootEvent: { id: "root-event-1", nodeId: "trigger-1" } }], event),
    ).toBe("run-1");
  });

  it("matches a run by execution id before falling back to root event id", () => {
    const event = {
      id: "execution-1",
      title: "Schedule",
      state: "success",
      isOpen: false,
      kind: "execution",
      executionId: "execution-1",
      triggerEventId: "root-event-1",
    } satisfies SidebarEvent;

    expect(
      findRunIdForSidebarEvent(
        [
          { id: "run-from-root", rootEvent: { id: "root-event-1", nodeId: "trigger-1" } },
          { id: "run-from-execution", executions: [{ id: "execution-1" }] },
        ],
        event,
      ),
    ).toBe("run-from-execution");
  });
});

describe("sidebar event mappers", () => {
  it("keeps trigger event run ids available for run selection", () => {
    const sidebarEvent = mapTriggerEventToSidebarEvent(
      {
        id: "event-1",
        nodeId: "trigger-1",
        runId: "run-1",
        createdAt: "2026-07-07T10:00:00Z",
        data: {},
      },
      makeComponentsNode({ id: "trigger-1", type: "TYPE_TRIGGER" }),
    );

    expect(sidebarEvent.runId).toBe("run-1");
  });

  it("keeps execution run ids available for run selection", () => {
    const [sidebarEvent] = mapExecutionsToSidebarEvents(
      [
        {
          id: "execution-1",
          nodeId: "action-1",
          runId: "run-1",
          createdAt: "2026-07-07T10:00:00Z",
          rootEvent: { id: "event-1", nodeId: "trigger-1", runId: "run-1", data: {} },
        },
      ],
      [
        makeComponentsNode({ id: "action-1", type: "TYPE_ACTION" }),
        makeComponentsNode({ id: "trigger-1", type: "TYPE_TRIGGER" }),
      ],
    );

    expect(sidebarEvent?.runId).toBe("run-1");
  });
});
