import { describe, expect, it } from "vitest";

import { getImageMapper } from "./get_image";
import { buildImageComponentCtx, buildImageDetailsCtx, buildImageOutput } from "./image_mapper_test_helpers";

describe("getImageMapper", () => {
  it("includes image name node metadata", () => {
    const props = getImageMapper.props(
      buildImageComponentCtx({
        componentName: "oci.getImage",
        configuration: { image: "ocid1.image.oc1..example" },
        metadata: { imageName: "golden-image" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "disc", label: "golden-image" })]),
    );
  });

  it("maps output to details and falls back to execution createdAt for action time", () => {
    const createdAt = new Date("2026-01-01T09:00:00Z").toISOString();
    const ctx = buildImageDetailsCtx({
      execution: {
        createdAt,
        outputs: {
          default: [
            buildImageOutput({
              image: {
                id: "ocid1.image.oc1..example",
                displayName: "golden-image",
                lifecycleState: "AVAILABLE",
                operatingSystem: "Ubuntu",
                launchMode: "NATIVE",
                timeCreated: "2026-01-01T07:59:00Z",
              },
            }),
          ],
        },
      },
    });

    const details = getImageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBe(new Date(createdAt).toLocaleString());
    expect(details["Image ID"]).toBeUndefined();
    expect(details["Display Name"]).toBe("golden-image");
    expect(details["State"]).toBe("AVAILABLE");
    expect(details["Operating System"]).toBe("Ubuntu");
    expect(details["Launch Mode"]).toBe("NATIVE");
  });
});
