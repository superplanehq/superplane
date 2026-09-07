import { beforeEach, describe, expect, it, vi } from "vitest";

const canvasesPutCanvasStaging = vi.hoisted(() => vi.fn());
const canvasesCommitCanvasStaging = vi.hoisted(() => vi.fn());

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    canvasesPutCanvasStaging,
    canvasesCommitCanvasStaging,
  };
});

import { stageAndCommitFactorySpecs } from "./installFactoryCanvas";

describe("stageAndCommitFactorySpecs", () => {
  beforeEach(() => {
    canvasesPutCanvasStaging.mockReset().mockResolvedValue({});
    canvasesCommitCanvasStaging.mockReset().mockResolvedValue({});
    window.history.replaceState(null, "", "/onboarding?attempt=attempt-1");
  });

  it("sends x-organization-id when the browser is on the unscoped onboarding route", async () => {
    await stageAndCommitFactorySpecs("github-owner", "canvas-1", "canvas: {}", "console: {}");

    expect(canvasesPutCanvasStaging).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: expect.objectContaining({ "x-organization-id": "github-owner" }),
      }),
    );
    expect(canvasesCommitCanvasStaging).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: expect.objectContaining({ "x-organization-id": "github-owner" }),
      }),
    );
  });
});
