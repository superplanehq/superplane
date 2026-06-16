import { describe, expect, it, vi } from "vitest";
import { makeComponentsNode } from "@/test/factories";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { findRunIdForSidebarEvent, getSidebarEventRunLookupBefore, mapCanvasNodesToLogEntries } from "./utils";

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

describe("getSidebarEventRunLookupBefore", () => {
  it("starts run lookup just after the root event timestamp", () => {
    const event = {
      id: "execution-1",
      title: "Schedule",
      state: "success",
      isOpen: false,
      kind: "execution",
      receivedAt: new Date("2026-02-06T15:28:05Z"),
      originalExecution: {
        id: "execution-1",
        rootEvent: {
          id: "root-event-1",
          createdAt: "2026-02-06T15:00:00Z",
        },
      },
    } satisfies SidebarEvent;

    expect(getSidebarEventRunLookupBefore(event)).toBe("2026-02-06T15:05:00.000Z");
  });
});
