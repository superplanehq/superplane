import { describe, expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";
import { copyImageMapper } from "./copy_image";

describe("copyImageMapper.getExecutionDetails", () => {
  it("formats the first default output with the configured source image in order", () => {
    const details = copyImageMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { sourceImageId: "ami-source" },
        metadata: { sourceImageId: "ami-metadata", imageId: "ami-metadata-copy" },
        outputs: {
          default: [
            {
              data: {
                image: {
                  imageId: "ami-copy",
                  name: "copied-image",
                  description: "Copied image",
                  state: "available",
                  creationDate: "2026-05-21T12:00:00Z",
                  architecture: "x86_64",
                  imageType: "machine",
                  rootDeviceType: "ebs",
                  rootDeviceName: "/dev/sda1",
                  virtualizationType: "hvm",
                  hypervisor: "xen",
                  ownerId: "111122223333",
                },
              },
            },
            { data: { image: { imageId: "ami-second" } } },
          ],
        },
      }),
    );

    expect(Object.entries(details)).toEqual([
      ["Source Image ID", "ami-source"],
      ["Image ID", "ami-copy"],
      ["Name", "copied-image"],
      ["Description", "Copied image"],
      ["State", "available"],
      ["Creation Date", "2026-05-21T12:00:00Z"],
      ["Architecture", "x86_64"],
      ["Image Type", "machine"],
      ["Root Device Type", "ebs"],
      ["Root Device Name", "/dev/sda1"],
      ["Virtualization Type", "hvm"],
      ["Hypervisor", "xen"],
      ["Owner ID", "111122223333"],
    ]);
  });

  it.each([
    {
      metadata: { imageId: "ami-pending", sourceImageId: "ami-metadata", sourceRegion: "us-east-1", state: "pending" },
      expected: [
        ["Image ID", "ami-pending"],
        ["Region", "us-west-2"],
        ["Source Image ID", "ami-metadata"],
        ["Source Region", "us-east-1"],
        ["State", "pending"],
      ],
    },
    {
      metadata: { imageId: undefined, sourceImageId: null, sourceRegion: "", state: "" },
      expected: [
        ["Image ID", "-"],
        ["Region", "us-west-2"],
        ["Source Image ID", "-"],
        ["Source Region", "-"],
        ["State", "-"],
      ],
    },
  ])("uses metadata and the configured region when output is missing: %j", ({ metadata, expected }) => {
    const details = copyImageMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { region: "us-west-2", sourceImageId: "ami-configuration", sourceRegion: "eu-west-1" },
        metadata: { region: "eu-central-1", ...metadata },
      }),
    );

    expect(Object.entries(details)).toEqual(expected);
  });

  it("keeps the output detail set when the output has no image", () => {
    const details = copyImageMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { sourceImageId: "ami-source" },
        metadata: { imageId: "ami-metadata-copy", sourceImageId: "ami-metadata", state: "pending" },
        outputs: { default: [{ data: {} }] },
      }),
    );

    expect(Object.entries(details)).toEqual([
      ["Source Image ID", "ami-source"],
      ["Image ID", "-"],
      ["Name", "-"],
      ["Description", "-"],
      ["State", "-"],
      ["Creation Date", "-"],
      ["Architecture", "-"],
      ["Image Type", "-"],
      ["Root Device Type", "-"],
      ["Root Device Name", "-"],
      ["Virtualization Type", "-"],
      ["Hypervisor", "-"],
      ["Owner ID", "-"],
    ]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Copy image",
    componentName: "aws.ec2.copyImage",
    isCollapsed: false,
    configuration: execution.configuration,
  };

  return {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
      ...execution,
    },
  };
}
