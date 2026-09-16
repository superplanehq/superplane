import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { useCanvasId } from "./useCanvasId";

function renderCanvasId(path: string, route: string) {
  return renderHook(() => useCanvasId(), {
    wrapper: ({ children }: { children: ReactNode }) => (
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path={route} element={children} />
        </Routes>
      </MemoryRouter>
    ),
  });
}

describe("useCanvasId", () => {
  it("reads the factory automation id", () => {
    const { result } = renderCanvasId(
      "/org-1/workspaces/acme/automations/auto-1",
      "/:organizationId/workspaces/:factoryKey/automations/:automationId",
    );

    expect(result.current).toBe("auto-1");
  });

  it("reads the app id", () => {
    const { result } = renderCanvasId("/org-1/apps/app-1", "/:organizationId/apps/:appId");

    expect(result.current).toBe("app-1");
  });
});
