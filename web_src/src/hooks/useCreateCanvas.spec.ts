import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const canvasesCreateCanvas = vi.hoisted(() => vi.fn());

vi.mock("../api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    canvasesCreateCanvas,
  };
});

import { useCreateCanvas } from "@/hooks/useCanvasData";

describe("useCreateCanvas", () => {
  beforeEach(() => {
    canvasesCreateCanvas.mockReset();
    canvasesCreateCanvas.mockResolvedValue({ data: { canvas: { metadata: { id: "canvas-1" } } } });
    window.history.replaceState(null, "", "/onboarding?attempt=attempt-1");
  });

  it("sends x-organization-id when the browser is on the unscoped onboarding route", async () => {
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children);

    const { result } = renderHook(() => useCreateCanvas("github-owner"), { wrapper });
    await result.current.mutateAsync({ name: "Plan", method: "ui" });

    expect(canvasesCreateCanvas).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: expect.objectContaining({ "x-organization-id": "github-owner" }),
      }),
    );
  });
});
