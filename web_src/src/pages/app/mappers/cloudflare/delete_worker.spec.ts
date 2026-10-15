import { describe, expect, it } from "bun:test";

import { deleteWorkerMapper } from "./delete_worker";
import { eventStateRegistry } from "./index";
import { buildExecution, createNodeFixtures } from "./test_support";

const { buildDetailsCtx } = createNodeFixtures("cloudflare.deleteWorker");

describe("deleteWorkerMapper.getExecutionDetails", () => {
  it("includes script name and deleted flag", () => {
    const data = { workerScript: "gone", deleted: true };
    const outputs = { default: [{ type: "cloudflare.worker.deleted", timestamp: new Date().toISOString(), data }] };
    const details = deleteWorkerMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs } }));
    expect(details["Script"]).toBe("gone");
    expect(details["Deleted"]).toBe("true");
  });
});

describe("eventStateRegistry.deleteWorker", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry.deleteWorker.getState(buildExecution())).toBe("deleted");
  });
});
