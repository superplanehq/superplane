import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type Dispatch, type ReactNode, type SetStateAction } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { fetchRepositoryFileContentCachedMock } = vi.hoisted(() => ({
  fetchRepositoryFileContentCachedMock: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  fetchRepositoryFileContentCached: fetchRepositoryFileContentCachedMock,
}));

import { useEditorLifecycle } from "./useEditorLifecycle";

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

type CommittedContent = Record<string, string>;

type HookArgs = {
  canvasId?: string;
  versionId?: string;
  selectedPath: string | null;
  setCommittedContentByPath: Dispatch<SetStateAction<CommittedContent>>;
};

function renderUseEditorLifecycle(args: HookArgs) {
  const queryClient = createQueryClient();
  return renderHook(
    ({ canvasId, versionId, selectedPath, setCommittedContentByPath }: HookArgs) =>
      useEditorLifecycle({
        canvasId,
        versionId,
        isEditing: true,
        resetPendingState: () => {},
        setIsDiffOpen: () => {},
        selectedPath,
        setLoadedContentByPath: () => {},
        setCommittedContentByPath,
        setHeaderActionsHost: () => {},
      }),
    {
      wrapper: createWrapper(queryClient),
      initialProps: args,
    },
  );
}

describe("useEditorLifecycle committed-content read", () => {
  let unhandledRejections: unknown[];
  let rejectionHandler: (event: PromiseRejectionEvent) => void;

  beforeEach(() => {
    unhandledRejections = [];
    rejectionHandler = (event: PromiseRejectionEvent) => {
      unhandledRejections.push(event.reason);
    };
    window.addEventListener("unhandledrejection", rejectionHandler);
  });

  afterEach(() => {
    window.removeEventListener("unhandledrejection", rejectionHandler);
    fetchRepositoryFileContentCachedMock.mockReset();
  });

  it("does not surface an unhandled rejection when the committed read fails", async () => {
    fetchRepositoryFileContentCachedMock.mockRejectedValue(new Error("Failed to get file"));

    const setCommittedContentByPath = vi.fn();

    renderUseEditorLifecycle({
      canvasId: "canvas-1",
      versionId: "version-1",
      selectedPath: "src/app.ts",
      setCommittedContentByPath,
    });

    await waitFor(() => {
      expect(fetchRepositoryFileContentCachedMock).toHaveBeenCalled();
    });

    // Give the rejection one more macrotask to flush before we assert no
    // unhandledrejection event was raised on window.
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(unhandledRejections).toEqual([]);
    expect(setCommittedContentByPath).toHaveBeenCalledTimes(1);
    const updater = setCommittedContentByPath.mock.calls[0][0] as (prev: CommittedContent) => CommittedContent;
    expect(updater({})).toEqual({ "src/app.ts": "" });
  });

  it("stores the committed content for the selected path when the read succeeds", async () => {
    fetchRepositoryFileContentCachedMock.mockResolvedValue("committed content");

    const setCommittedContentByPath = vi.fn();

    renderUseEditorLifecycle({
      canvasId: "canvas-1",
      versionId: "version-1",
      selectedPath: "src/app.ts",
      setCommittedContentByPath,
    });

    await waitFor(() => {
      expect(setCommittedContentByPath).toHaveBeenCalledTimes(1);
    });

    const updater = setCommittedContentByPath.mock.calls[0][0] as (prev: CommittedContent) => CommittedContent;
    expect(updater({})).toEqual({ "src/app.ts": "committed content" });
    expect(unhandledRejections).toEqual([]);
  });
});
