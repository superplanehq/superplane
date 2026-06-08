import { describe, expect, it } from "vitest";
import { deleteImageMapper } from "./delete_image";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("deleteImageMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteImageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns only Executed At when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    const details = deleteImageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Image Name"]).toBeUndefined();
  });

  it("extracts the deleted image name and marks it deleted", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ imageName: "my-image" })],
        },
      },
    });
    const details = deleteImageMapper.getExecutionDetails(ctx);
    expect(details["Image Name"]).toBe("my-image");
    expect(details["Status"]).toBe("Deleted");
  });
});
