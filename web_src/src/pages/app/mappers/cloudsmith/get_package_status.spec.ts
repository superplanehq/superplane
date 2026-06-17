import { describe, expect, it } from "vitest";
import { getPackageStatusMapper } from "./get_package_status";
import { buildDetailsCtx, buildPackageOutput, buildPackageStatusData } from "./test_helpers";

describe("getPackageStatusMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getPackageStatusMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getPackageStatusMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At without status fields when output data is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(undefined, "cloudsmith.package.status")] } },
    });
    const details = getPackageStatusMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Stage"]).toBeUndefined();
  });

  it("extracts stage, status, sync and quarantine fields from the payload", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildPackageOutput(buildPackageStatusData(), "cloudsmith.package.status")] },
      },
    });
    const details = getPackageStatusMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Stage"]).toBe("Available");
    expect(details["Status"]).toBe("Available");
    expect(details["Sync Progress"]).toBe("100%");
    expect(details["Sync Completed"]).toBe("Yes");
    expect(details["Quarantined"]).toBe("No");
  });

  it("shows sync progress as 0% when sync has not started", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildPackageOutput(
              buildPackageStatusData({ sync_progress: 0, is_sync_completed: false, is_sync_in_progress: true }),
              "cloudsmith.package.status",
            ),
          ],
        },
      },
    });
    const details = getPackageStatusMapper.getExecutionDetails(ctx);
    expect(details["Sync Progress"]).toBe("0%");
    expect(details["Sync Completed"]).toBe("No");
  });

  it("shows Quarantined as Yes when package is quarantined", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildPackageOutput(buildPackageStatusData({ is_quarantined: true }), "cloudsmith.package.status")],
        },
      },
    });
    const details = getPackageStatusMapper.getExecutionDetails(ctx);
    expect(details["Quarantined"]).toBe("Yes");
  });
});
