import { beforeEach, describe, expect, it, vi } from "bun:test";

import { unmockedSrc } from "@/test/unmockedModule";

const canvasesPutCanvasStaging = vi.hoisted(() => vi.fn());
const canvasesCommitCanvasStaging = vi.hoisted(() => vi.fn());

vi.mock("@/api-client", () => {
  const actual = unmockedSrc<Record<string, unknown>>("api-client");
  return {
    ...actual,
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
