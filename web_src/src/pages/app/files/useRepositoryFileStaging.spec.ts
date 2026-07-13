import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { PendingFileChange } from "./types";
import { useRepositoryFileStaging } from "./useRepositoryFileStaging";

const mutateAsyncStage = vi.fn();
const mutateAsyncDiscard = vi.fn();

vi.mock("@/hooks/useCanvasData", () => ({
  useStageRepositoryFiles: () => ({
    mutateAsync: mutateAsyncStage,
  }),
  useDiscardRepositoryFilePaths: () => ({
    mutateAsync: mutateAsyncDiscard,
  }),
}));

vi.mock("../lib/staging-content-match", () => ({
  matchesCommittedRepositoryFileContent: vi.fn(async () => false),
}));

const pendingChange: PendingFileChange = {
  type: "modified",
  path: "test.sh",
  content: "#!/bin/bash\necho edited\n",
};

describe("useRepositoryFileStaging", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mutateAsyncStage.mockResolvedValue(undefined);
    mutateAsyncDiscard.mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
  });

  it("does not discard server staging when edit mode is toggled off and back on", async () => {
    const { rerender } = renderHook(
      ({ enabled, pendingChanges }: { enabled: boolean; pendingChanges: PendingFileChange[] }) =>
        useRepositoryFileStaging({
          canvasId: "canvas-1",
          versionId: "version-1",
          enabled,
          pendingChanges,
        }),
      {
        initialProps: {
          enabled: true,
          pendingChanges: [pendingChange],
        },
      },
    );

    await act(async () => {
      vi.advanceTimersByTime(500);
      await Promise.resolve();
    });

    expect(mutateAsyncStage).toHaveBeenCalledTimes(1);
    expect(mutateAsyncDiscard).not.toHaveBeenCalled();

    rerender({ enabled: false, pendingChanges: [] });

    await act(async () => {
      vi.advanceTimersByTime(500);
      await Promise.resolve();
    });

    expect(mutateAsyncDiscard).not.toHaveBeenCalled();

    rerender({ enabled: true, pendingChanges: [] });

    await act(async () => {
      vi.advanceTimersByTime(500);
      await Promise.resolve();
    });

    expect(mutateAsyncDiscard).not.toHaveBeenCalled();
  });
});
