import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { RunsSidebarHrefProvider } from "./runsSidebarHref";
import { useRunsSidebarRunHref } from "./useRunsSidebarRunHref";

describe("useRunsSidebarRunHref", () => {
  it("uses the factory href when a provider supplies one", () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <RunsSidebarHrefProvider hrefForRun={(runId) => `/factory?run=${runId}`}>{children}</RunsSidebarHrefProvider>
    );

    const { result } = renderHook(() => useRunsSidebarRunHref("run-1", "/org/apps/app-1?run=run-1"), { wrapper });

    expect(result.current).toBe("/factory?run=run-1");
  });

  it("falls back to the canvas route when no provider is set", () => {
    const { result } = renderHook(() => useRunsSidebarRunHref("run-1", "/org/apps/app-1?run=run-1"));

    expect(result.current).toBe("/org/apps/app-1?run=run-1");
  });
});
