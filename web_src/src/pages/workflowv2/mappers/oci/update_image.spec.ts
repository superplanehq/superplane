import { describe, expect, it } from "vitest";

import { updateImageMapper } from "./update_image";
import { buildImageComponentCtx, buildImageDetailsCtx, buildImageOutput } from "./image_mapper_test_helpers";

describe("updateImageMapper", () => {
  it("includes image name and target display name metadata", () => {
    const props = updateImageMapper.props(
      buildImageComponentCtx({
        componentName: "oci.updateImage",
        configuration: { image: "ocid1.image.oc1..example", displayName: "production-image" },
        metadata: { imageName: "golden-image" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "disc", label: "golden-image" }),
        expect.objectContaining({ icon: "tag", label: "production-image" }),
      ]),
    );
  });

  it("maps updated image output to details", () => {
    const ctx = buildImageDetailsCtx({
      execution: {
        metadata: { startedAt: "2026-01-01T08:00:00Z" },
        outputs: {
          default: [
            buildImageOutput({
              image: {
                id: "ocid1.image.oc1..example",
                displayName: "production-image",
                lifecycleState: "AVAILABLE",
                operatingSystem: "Oracle Linux",
                operatingSystemVersion: "9",
                launchMode: "PARAVIRTUALIZED",
                timeCreated: "2026-01-01T07:59:00Z",
              },
            }),
          ],
        },
      },
    });

    const details = updateImageMapper.getExecutionDetails(ctx);
    expect(details["Display Name"]).toBe("production-image");
    expect(details["State"]).toBe("AVAILABLE");
    expect(details["Operating System"]).toBe("Oracle Linux 9");
    expect(details["Executed At"]).toBe(new Date("2026-01-01T08:00:00Z").toLocaleString());
  });
});
