import { describe, expect, it, vi } from "vitest";

import { applySplitRunStop, applySplitRunStopChoice } from "./splitRunStop";

describe("applySplitRunStopChoice", () => {
  it("closes as rejected for Stop as Canceled", async () => {
    const onClose = vi.fn();
    const onStatusChange = vi.fn();

    await applySplitRunStopChoice("canceled", { onClose, onStatusChange });

    expect(onClose).toHaveBeenCalledWith("RESULT_REJECTED");
    expect(onStatusChange).not.toHaveBeenCalled();
  });

  it("closes as completed for Stop as Completed", async () => {
    const onClose = vi.fn();
    const onStatusChange = vi.fn();

    await applySplitRunStopChoice("completed", { onClose, onStatusChange });

    expect(onClose).toHaveBeenCalledWith("RESULT_COMPLETED");
    expect(onStatusChange).not.toHaveBeenCalled();
  });

  it("reruns the current step", async () => {
    const onClose = vi.fn();
    const onStatusChange = vi.fn();
    const onRerun = vi.fn();

    await applySplitRunStopChoice("rerun-step", { onClose, onStatusChange, onRerun });

    expect(onRerun).toHaveBeenCalledWith("rerun-step");
    expect(onClose).not.toHaveBeenCalled();
    expect(onStatusChange).not.toHaveBeenCalled();
  });

  it("reruns from the first step", async () => {
    const onClose = vi.fn();
    const onStatusChange = vi.fn();
    const onRerun = vi.fn();

    await applySplitRunStopChoice("rerun-start", { onClose, onStatusChange, onRerun });

    expect(onRerun).toHaveBeenCalledWith("rerun-start");
    expect(onClose).not.toHaveBeenCalled();
    expect(onStatusChange).not.toHaveBeenCalled();
  });

  it("reopens for the Reopen outcome", async () => {
    const onClose = vi.fn();
    const onStatusChange = vi.fn();

    await applySplitRunStopChoice("reopen", { onClose, onStatusChange });

    expect(onStatusChange).toHaveBeenCalledWith("STATE_OPEN");
    expect(onClose).not.toHaveBeenCalled();
  });
});

describe("applySplitRunStop", () => {
  const run = { appId: "app-implement", runId: "run-1" };

  it("cancels the canvas run before closing a running order", async () => {
    const cancelRun = vi.fn().mockResolvedValue(undefined);
    const onClose = vi.fn();
    const onStatusChange = vi.fn();

    await applySplitRunStop("canceled", {
      kind: "running",
      run,
      cancelRun,
      onClose,
      onStatusChange,
    });

    expect(cancelRun.mock.invocationCallOrder[0]).toBeLessThan(onClose.mock.invocationCallOrder[0]);
    expect(cancelRun).toHaveBeenCalledWith(run);
    expect(onClose).toHaveBeenCalledWith("RESULT_REJECTED");
  });

  it("does not cancel when the order is waiting", async () => {
    const cancelRun = vi.fn();
    const onClose = vi.fn();

    await applySplitRunStop("completed", {
      kind: "waiting",
      run,
      cancelRun,
      onClose,
      onStatusChange: vi.fn(),
    });

    expect(cancelRun).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledWith("RESULT_COMPLETED");
  });

  it("does not cancel when the order failed", async () => {
    const cancelRun = vi.fn();
    const onStatusChange = vi.fn();

    const onRerun = vi.fn();
    await applySplitRunStop("rerun-step", {
      kind: "failed",
      run,
      cancelRun,
      onClose: vi.fn(),
      onStatusChange,
      onRerun,
    });

    expect(cancelRun).not.toHaveBeenCalled();
    expect(onRerun).toHaveBeenCalledWith("rerun-step");
    expect(onStatusChange).not.toHaveBeenCalled();
  });

  it("cancels a running step before rerunning it", async () => {
    const cancelRun = vi.fn().mockResolvedValue(undefined);
    const onRerun = vi.fn();

    await applySplitRunStop("rerun-step", {
      kind: "running",
      run,
      cancelRun,
      onClose: vi.fn(),
      onStatusChange: vi.fn(),
      onRerun,
    });

    expect(cancelRun.mock.invocationCallOrder[0]).toBeLessThan(onRerun.mock.invocationCallOrder[0]);
    expect(onRerun).toHaveBeenCalledWith("rerun-step");
  });

  it("still closes when a running order has no run id", async () => {
    const cancelRun = vi.fn();
    const onClose = vi.fn();

    await applySplitRunStop("canceled", {
      kind: "running",
      cancelRun,
      onClose,
      onStatusChange: vi.fn(),
    });

    expect(cancelRun).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledWith("RESULT_REJECTED");
  });

  it("does not change status when cancel fails", async () => {
    const cancelRun = vi.fn().mockRejectedValue(new Error("run still busy"));
    const onClose = vi.fn();

    await expect(
      applySplitRunStop("canceled", {
        kind: "running",
        run,
        cancelRun,
        onClose,
        onStatusChange: vi.fn(),
      }),
    ).rejects.toThrow("run still busy");

    expect(onClose).not.toHaveBeenCalled();
  });
});
