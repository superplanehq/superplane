import { describe, expect, it } from "vitest";
import { createImageMapper } from "./create_image";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("createImageMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createImageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns only Executed At when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    const details = createImageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Image Name"]).toBeUndefined();
  });

  it("extracts the created image fields", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "my-app-2026-06-02",
              family: "my-app",
              status: "READY",
              diskSizeGb: 10,
              sourceDisk: "my-disk",
            }),
          ],
        },
      },
    });
    const details = createImageMapper.getExecutionDetails(ctx);
    expect(details["Image Name"]).toBe("my-app-2026-06-02");
    expect(details["Family"]).toBe("my-app");
    expect(details["Status"]).toBe("READY");
    expect(details["Disk Size"]).toBe("10 GB");
    expect(details["Source Disk"]).toBe("my-disk");
  });
});
