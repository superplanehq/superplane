import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const { useWorkOrderEventsMock, useWorkOrderArtifactsMock, useFactoryPullRequestsMock } = vi.hoisted(() => ({
  useWorkOrderEventsMock: vi.fn(),
  useWorkOrderArtifactsMock: vi.fn(),
  useFactoryPullRequestsMock: vi.fn(),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useWorkOrderEvents: useWorkOrderEventsMock,
  useWorkOrderArtifacts: useWorkOrderArtifactsMock,
  useFactoryPullRequests: useFactoryPullRequestsMock,
}));

import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";

afterEach(() => {
  vi.clearAllMocks();
});

describe("useSplitRunStreamArtifacts", () => {
  it("indexes an artifact event by nodeId and overlays live data", () => {
    useWorkOrderEventsMock.mockReturnValue({
      data: {
        pages: [
          {
            events: [
              {
                type: "order.artifact.added",
                timestamp: "2026-08-24T16:32:18.000Z",
                event: {
                  automation: { nodeId: "add-pr" },
                  run: { id: "run-1" },
                  artifact: { id: "art-branch-1", type: "branch", data: { name: "feature/retry" } },
                },
              },
            ],
          },
        ],
      },
    });
    useWorkOrderArtifactsMock.mockReturnValue({
      data: [{ id: "art-branch-1", type: "TYPE_BRANCH", data: { name: "feature/retry-v2" } }],
    });
    useFactoryPullRequestsMock.mockReturnValue({ data: [] });

    const { result } = renderHook(() => useSplitRunStreamArtifacts("org-1", "factory-1", "order-1"));

    expect(result.current.byRun.get("run-1")?.byNodeId.get("add-pr")).toEqual({
      id: "art-branch-1",
      type: "TYPE_BRANCH",
      data: { name: "feature/retry-v2" },
    });
  });

  it("returns an empty index when the order id is missing", () => {
    useWorkOrderEventsMock.mockReturnValue({ data: { pages: [] } });
    useWorkOrderArtifactsMock.mockReturnValue({ data: [] });
    useFactoryPullRequestsMock.mockReturnValue({ data: [] });

    const { result } = renderHook(() => useSplitRunStreamArtifacts("org-1", "factory-1", undefined));

    expect(result.current.byRun.size).toBe(0);
  });
});
