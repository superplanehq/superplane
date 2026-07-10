import { describe, expect, it, vi } from "vitest";
import { selectCreatedRerun } from "./runInspectionRerunSelection";

describe("selectCreatedRerun", () => {
  it("selects the created rerun when it is available", async () => {
    const fetchRunId = vi.fn().mockResolvedValue("new-run-1");
    const selectRun = vi.fn();

    await selectCreatedRerun({
      eventId: "rerun-event-1",
      triggerNodeId: "trigger-node",
      selectedNodeId: "selected-node",
      fetchRunId,
      selectRun,
      attempts: 1,
      retryDelayMs: 0,
    });

    expect(fetchRunId).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "rerun-event-1",
        nodeId: "trigger-node",
        triggerEventId: "rerun-event-1",
      }),
      { maxPages: 1 },
    );
    expect(selectRun).toHaveBeenCalledWith("new-run-1", { nodeId: "selected-node" });
  });

  it("retries while the created rerun is not visible yet", async () => {
    const fetchRunId = vi.fn().mockResolvedValueOnce(null).mockResolvedValueOnce("new-run-1");
    const selectRun = vi.fn();

    await selectCreatedRerun({
      eventId: "rerun-event-1",
      triggerNodeId: "trigger-node",
      fetchRunId,
      selectRun,
      attempts: 2,
      retryDelayMs: 0,
    });

    expect(fetchRunId).toHaveBeenCalledTimes(2);
    expect(selectRun).toHaveBeenCalledWith("new-run-1", { nodeId: "trigger-node" });
  });
});
