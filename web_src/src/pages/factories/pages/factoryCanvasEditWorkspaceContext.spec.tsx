import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { FactoryCanvasEditWorkspaceProvider } from "./FactoryCanvasEditWorkspaceProvider";
import { useFactoryCanvasEditWorkspace } from "./factoryCanvasEditWorkspaceContext";

describe("useFactoryCanvasEditWorkspace", () => {
  it("is off outside the Storybook provider", () => {
    const { result } = renderHook(() => useFactoryCanvasEditWorkspace());
    expect(result.current).toBe(false);
  });

  it("is on inside the Storybook provider", () => {
    const { result } = renderHook(() => useFactoryCanvasEditWorkspace(), {
      wrapper: FactoryCanvasEditWorkspaceProvider,
    });
    expect(result.current).toBe(true);
  });
});
