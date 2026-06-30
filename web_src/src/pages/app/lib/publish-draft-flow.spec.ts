import type { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { ensureDraftVersionExists } from "@/hooks/useCanvasData";

import { publishDraftVersionAndExit } from "./publish-draft-flow";

vi.mock("@/hooks/useCanvasData", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    ensureDraftVersionExists: vi.fn(),
  };
});

describe("publishDraftVersionAndExit", () => {
  it("uses the version id captured after saves settle for publish and failure recovery", async () => {
    vi.mocked(ensureDraftVersionExists).mockResolvedValue(true);

    const activeCanvasVersionIdRef = { current: "draft-1" };
    const ensureVersionActionDraftReady = vi.fn().mockImplementation(async () => {
      activeCanvasVersionIdRef.current = "draft-2";
      return true;
    });
    const publishError = new Error("publish failed");
    const publishCanvasVersionMutation = {
      mutateAsync: vi.fn().mockRejectedValue(publishError),
    };

    const result = await publishDraftVersionAndExit({
      organizationId: "org-1",
      canvasId: "canvas-1",
      activeCanvasVersionIdRef,
      queryClient: {} as QueryClient,
      ensureVersionActionDraftReady,
      publishCanvasVersionMutation,
      runExitDraftToLive: vi.fn(),
      recoverFromMissingDraft: vi.fn(),
    });

    expect(publishCanvasVersionMutation.mutateAsync).toHaveBeenCalledWith({
      versionId: "draft-2",
      commitMessage: undefined,
    });
    expect(result).toEqual({
      status: "failed",
      versionIdToPublish: "draft-2",
      error: publishError,
    });
  });
});
