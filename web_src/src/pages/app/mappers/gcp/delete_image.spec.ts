import { describe, expect, it } from "bun:test";
import { deleteImageMapper } from "./delete_image";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("deleteImageMapper.getExecutionDetails", () => {
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
