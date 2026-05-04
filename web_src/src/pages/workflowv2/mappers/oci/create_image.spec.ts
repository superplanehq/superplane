import { describe, expect, it } from "vitest";

import { createImageMapper } from "./create_image";
import { buildImageComponentCtx, buildImageDetailsCtx, buildImageOutput } from "./image_mapper_test_helpers";

describe("createImageMapper.props", () => {
  it("includes human-readable node metadata", () => {
    const props = createImageMapper.props(
      buildImageComponentCtx({
        componentName: "oci.createImage",
        configuration: {
          displayName: "golden-image",
          compartment: "ocid1.compartment.oc1..example",
          sourceType: "instance",
        },
        metadata: {
          compartmentName: "Production",
          instanceName: "source-instance",
          sourceType: "instance",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "tag", label: "golden-image" }),
        expect.objectContaining({ icon: "folder", label: "Production" }),
        expect.objectContaining({ icon: "server", label: "source-instance" }),
      ]),
    );
  });

  it("limits node metadata to three items", () => {
    const props = createImageMapper.props(
      buildImageComponentCtx({
        componentName: "oci.createImage",
        configuration: {
          displayName: "golden-image",
          bucket: "images",
          object: "golden-image.qcow2",
          sourceType: "objectStorageObject",
        },
        metadata: {
          imageName: "created-image",
          compartmentName: "Production",
          instanceName: "source-instance",
        },
      }),
    );

    expect(props.metadata).toHaveLength(3);
  });
});

describe("createImageMapper.getExecutionDetails", () => {
  it("shows executed time and useful image details", () => {
    const startedAt = new Date("2026-01-01T08:00:00Z").toISOString();
    const ctx = buildImageDetailsCtx({
      execution: {
        metadata: { startedAt },
        outputs: {
          default: [
            buildImageOutput({
              image: {
                id: "ocid1.image.oc1..example",
                displayName: "golden-image",
                lifecycleState: "AVAILABLE",
                operatingSystem: "Oracle Linux",
                operatingSystemVersion: "8",
                launchMode: "PARAVIRTUALIZED",
                timeCreated: "2026-01-01T07:59:00Z",
              },
            }),
          ],
        },
      },
    });

    const details = createImageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBe(new Date(startedAt).toLocaleString());
    expect(details["Image ID"]).toBeUndefined();
    expect(details["Display Name"]).toBe("golden-image");
    expect(details["State"]).toBe("AVAILABLE");
    expect(details["Operating System"]).toBe("Oracle Linux 8");
    expect(details["Launch Mode"]).toBe("PARAVIRTUALIZED");
    expect(details["Created At"]).toBeUndefined();
  });

  it("does not throw when outputs are missing", () => {
    const ctx = buildImageDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createImageMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(createImageMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});
