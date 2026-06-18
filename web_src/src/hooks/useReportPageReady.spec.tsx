import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

const { sendPageObservabilityReady } = vi.hoisted(() => ({
  sendPageObservabilityReady: vi.fn(),
}));

vi.mock("@/lib/dash0Observability", () => ({
  sendPageObservabilityReady,
}));

import { useReportPageReady } from "@/hooks/useReportPageReady";

function createWrapper(path: string) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(
      MemoryRouter,
      { initialEntries: [path] },
      createElement(Routes, null, createElement(Route, { path: "/:organizationId", element: children })),
    );
  };
}

describe("useReportPageReady", () => {
  beforeEach(() => {
    sendPageObservabilityReady.mockClear();
  });

  it("sends ready once when ready becomes true", async () => {
    const { rerender } = renderHook(({ ready }) => useReportPageReady(ready, { canvas_count: 2 }), {
      wrapper: createWrapper("/org-1"),
      initialProps: { ready: false },
    });

    expect(sendPageObservabilityReady).not.toHaveBeenCalled();

    rerender({ ready: true });

    await waitFor(() => {
      expect(sendPageObservabilityReady).toHaveBeenCalledWith(
        "organizationHomePage",
        expect.objectContaining({
          organization_id: "org-1",
          canvas_count: 2,
        }),
      );
    });

    rerender({ ready: true });
    expect(sendPageObservabilityReady).toHaveBeenCalledTimes(1);
  });
});
