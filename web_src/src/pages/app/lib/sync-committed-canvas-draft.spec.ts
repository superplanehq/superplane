import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { canvasesDescribeCanvas } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { syncCommittedCanvasDraftState } from "./sync-committed-canvas-draft";

vi.mock("@/api-client", async () => {
  const actual = await vi.importActual<typeof import("@/api-client")>("@/api-client");
  return {
    ...actual,
    canvasesDescribeCanvas: vi.fn(),
  };
});

vi.mock("@/hooks/useCanvasData", async () => {
  const actual = await vi.importActual<typeof import("@/hooks/useCanvasData")>("@/hooks/useCanvasData");
  return {
    ...actual,
    ensureCanvasVersion: vi.fn(),
  };
});

import { ensureCanvasVersion } from "@/hooks/useCanvasData";

describe("syncCommittedCanvasDraftState", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("loads a specific committed version through ensureCanvasVersion", async () => {
    const committedVersion = {
      metadata: { id: "version-1" },
      spec: { nodes: [], edges: [] },
    };
    vi.mocked(ensureCanvasVersion).mockResolvedValue(committedVersion);

    const setQueryData = vi.fn();
    const queryClient = { setQueryData } as unknown as QueryClient;

    const result = await syncCommittedCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "version-1",
    });

    expect(result).toEqual(committedVersion);
    expect(ensureCanvasVersion).toHaveBeenCalledWith(queryClient, "canvas-1", "version-1");
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.versionDetail("canvas-1", "version-1"), committedVersion);
  });

  it("loads the live committed version from DescribeCanvas when resolveLiveVersion is true", async () => {
    const committedVersion = {
      metadata: { id: "live-version-2", canvasId: "canvas-1" },
      spec: { nodes: [{ id: "node-1" }], edges: [] },
    };
    vi.mocked(canvasesDescribeCanvas).mockResolvedValue({
      data: {
        canvas: {
          metadata: { id: "canvas-1", versionId: "live-version-2" },
          spec: committedVersion.spec,
        },
      },
    } as never);

    const setQueryData = vi.fn();
    const queryClient = { setQueryData } as unknown as QueryClient;

    const result = await syncCommittedCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "version-1",
      resolveLiveVersion: true,
    });

    expect(result).toEqual(committedVersion);
    expect(ensureCanvasVersion).not.toHaveBeenCalled();
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.versionDetail("canvas-1", "live-version-2"), committedVersion);
  });
});

describe("syncCommittedConsoleCaches", () => {
  it("mirrors committed console data into version and staged caches", async () => {
    const committedVersion = {
      metadata: { id: "version-1" },
      spec: {
        panels: [{ id: "panel-1", type: "markdown", content: { title: "Hello" } }],
        layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
      },
    };

    const fetchQuery = vi.fn().mockResolvedValue(committedVersion);
    const setQueryData = vi.fn();
    const queryClient = { fetchQuery, setQueryData } as unknown as QueryClient;

    const { syncCommittedConsoleCaches } = await import("./sync-committed-canvas-draft");
    await syncCommittedConsoleCaches({
      queryClient,
      canvasId: "canvas-1",
      versionId: "version-1",
    });

    expect(fetchQuery).toHaveBeenCalled();
    expect(setQueryData).toHaveBeenCalledWith(
      canvasKeys.console("canvas-1", "version-1"),
      expect.objectContaining({
        canvasId: "canvas-1",
        versionId: "version-1",
        panels: committedVersion.spec.panels,
        layout: committedVersion.spec.layout,
      }),
    );
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.stagedConsole("canvas-1"), expect.any(Object));
  });
});
