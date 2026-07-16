import { describe, expect, it, vi } from "vitest";
import { runSettingsTabAutosave } from "./settingsTabAutosaveActions";

describe("runSettingsTabAutosave", () => {
  it("queues pending saves while a save is already in flight", async () => {
    const queuePendingAutosave = vi.fn();
    const flushPendingAutosave = vi.fn();
    const savingRef = { current: true };

    await runSettingsTabAutosave({
      baselineSnapshot: "{}",
      currentNodeName: "Node",
      flushPendingAutosave,
      nodeConfiguration: { enabled: true },
      onSave: vi.fn(),
      queuePendingAutosave,
      savingRef,
      updateAutosaveBaseline: vi.fn(),
      validateNow: vi.fn(),
    });

    expect(queuePendingAutosave).toHaveBeenCalledTimes(1);
    expect(flushPendingAutosave).not.toHaveBeenCalled();
  });

  it("flushes pending saves after a rejected async save", async () => {
    const flushPendingAutosave = vi.fn();
    const savingRef = { current: false };
    const onSave = vi.fn().mockRejectedValue(new Error("save failed"));

    await expect(
      runSettingsTabAutosave({
        baselineSnapshot: "{}",
        currentNodeName: "Node",
        flushPendingAutosave,
        nodeConfiguration: { enabled: true },
        onSave,
        queuePendingAutosave: vi.fn(),
        savingRef,
        updateAutosaveBaseline: vi.fn(),
        validateNow: vi.fn(),
      }),
    ).rejects.toThrow("save failed");

    expect(savingRef.current).toBe(false);
    expect(flushPendingAutosave).toHaveBeenCalledTimes(1);
  });
});
