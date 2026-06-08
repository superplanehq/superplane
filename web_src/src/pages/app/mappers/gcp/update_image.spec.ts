import { describe, expect, it } from "vitest";
import { updateImageMapper } from "./update_image";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("updateImageMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => updateImageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns only Executed At when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    const details = updateImageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Deprecation State"]).toBeUndefined();
  });

  it("maps the deprecation state to a friendly label", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "my-image",
              family: "my-app",
              deprecationState: "DEPRECATED",
              replacement: "my-app-v2",
            }),
          ],
        },
      },
    });
    const details = updateImageMapper.getExecutionDetails(ctx);
    expect(details["Image Name"]).toBe("my-image");
    expect(details["Deprecation State"]).toBe("Deprecated");
    expect(details["Replacement"]).toBe("my-app-v2");
  });
});
