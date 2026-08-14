import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useFactoryConfigureInitialLayout } from "./useFactoryConfigureInitialLayout";

describe("useFactoryConfigureInitialLayout", () => {
  it("applies layout once when Factory Configure becomes ready", async () => {
    const applyLayout = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      (props: { factoryAutoLayout: boolean; isEditing: boolean; editBootstrapReady: boolean }) =>
        useFactoryConfigureInitialLayout({
          ...props,
          activeCanvasVersionId: "version-live",
          applyLayout,
        }),
      {
        initialProps: { factoryAutoLayout: true, isEditing: false, editBootstrapReady: false },
      },
    );

    expect(applyLayout).not.toHaveBeenCalled();

    await act(async () => {
      rerender({ factoryAutoLayout: true, isEditing: true, editBootstrapReady: true });
    });
    expect(applyLayout).toHaveBeenCalledTimes(1);

    await act(async () => {
      rerender({ factoryAutoLayout: true, isEditing: true, editBootstrapReady: true });
    });
    expect(applyLayout).toHaveBeenCalledTimes(1);
  });

  it("applies layout again on a new Configure visit", async () => {
    const applyLayout = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ factoryAutoLayout }: { factoryAutoLayout: boolean }) =>
        useFactoryConfigureInitialLayout({
          factoryAutoLayout,
          isEditing: true,
          editBootstrapReady: true,
          activeCanvasVersionId: "version-live",
          applyLayout,
        }),
      { initialProps: { factoryAutoLayout: true } },
    );

    expect(applyLayout).toHaveBeenCalledTimes(1);

    await act(async () => rerender({ factoryAutoLayout: false }));
    await act(async () => rerender({ factoryAutoLayout: true }));

    expect(applyLayout).toHaveBeenCalledTimes(2);
  });
});
