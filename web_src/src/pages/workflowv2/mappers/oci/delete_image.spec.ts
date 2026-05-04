import { describe, expect, it } from "vitest";

import { deleteImageMapper } from "./delete_image";
import { buildImageComponentCtx, buildImageDetailsCtx, buildImageOutput } from "./image_mapper_test_helpers";

describe("deleteImageMapper", () => {
  it("includes image name metadata", () => {
    const props = deleteImageMapper.props(
      buildImageComponentCtx({
        componentName: "oci.deleteImage",
        configuration: { image: "ocid1.image.oc1..example" },
        metadata: { imageName: "golden-image" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "disc", label: "golden-image" })]),
    );
  });

  it("maps deletion output to details with action time", () => {
    const ctx = buildImageDetailsCtx({
      execution: {
        metadata: { startedAt: "2026-01-01T08:00:00Z" },
        outputs: {
          default: [
            buildImageOutput({
              imageId: "ocid1.image.oc1..example",
              state: "DELETED",
              deletedAt: "2026-01-01T08:00:01Z",
            }),
          ],
        },
      },
    });

    const details = deleteImageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBe(new Date("2026-01-01T08:00:00Z").toLocaleString());
    expect(details["Image ID"]).toBeUndefined();
    expect(details["State"]).toBe("DELETED");
    expect(details["Deleted At"]).toBeUndefined();
  });
});
