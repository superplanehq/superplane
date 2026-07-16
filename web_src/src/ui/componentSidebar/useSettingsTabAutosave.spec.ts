import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useSettingsTabAutosave } from "./useSettingsTabAutosave";

describe("useSettingsTabAutosave", () => {
  it("flushes pending edits when interaction becomes disabled", async () => {
    const onSave = vi.fn();
    const validateNow = vi.fn();

    const { rerender } = renderHook(
      ({ isInteractionDisabled, nodeConfiguration }) =>
        useSettingsTabAutosave({
          currentNodeName: "Node",
          initialConfiguration: {},
          initialNodeName: "Node",
          isInteractionDisabled,
          nodeConfiguration,
          onSave,
          validateNow,
        }),
      {
        initialProps: {
          isInteractionDisabled: false,
          nodeConfiguration: { enabled: true },
        },
      },
    );

    rerender({
      isInteractionDisabled: true,
      nodeConfiguration: { enabled: true },
    });

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledWith({ enabled: true }, "Node", undefined);
    });
  });
});
