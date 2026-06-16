import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasRun } from "@/api-client";
import type { SidebarEvent } from "@/ui/componentSidebar/types";

const { canvasesListRuns } = vi.hoisted(() => ({
  canvasesListRuns: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api-client")>();
  return {
    ...actual,
    canvasesListRuns,
  };
});

import { useSidebarEventRunLookup } from "./useSidebarEventRunLookup";

const triggerEvent = {
  id: "root-1",
  title: "Trigger",
  state: "success",
  isOpen: false,
  kind: "trigger",
} satisfies SidebarEvent;

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useSidebarEventRunLookup", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    canvasesListRuns.mockResolvedValue({ data: { runs: [] } });
  });

  it("does not cache failed paginated lookups so later cached runs can resolve", async () => {
    const queryClient = new QueryClient();
    const runsWithMatch: CanvasesCanvasRun[] = [
      {
        id: "run-1",
        rootEvent: { id: "root-1" },
      },
    ];

    const { result, rerender } = renderHook(
      (props: { runs: CanvasesCanvasRun[] }) =>
        useSidebarEventRunLookup({
          enabled: true,
          canvasId: "canvas-1",
          organizationId: "org-1",
          queryClient,
          runs: props.runs,
        }),
      {
        wrapper: createWrapper(queryClient),
        initialProps: { runs: [] },
      },
    );

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(triggerEvent)).toBeNull();
    });

    rerender({ runs: runsWithMatch });

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(triggerEvent)).toBe("run-1");
    });

    expect(canvasesListRuns).toHaveBeenCalledTimes(1);
  });
});
