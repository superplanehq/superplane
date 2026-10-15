import { describe, expect, it } from "bun:test";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";
import { createImageMapper } from "./create_image";

describe("createImageMapper.getExecutionDetails", () => {
  it("keeps the success-path fields in order", () => {
    const details = createImageMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { instanceId: "i-123" },
        outputs: {
          default: [
            {
              data: {
                image: {
                  imageId: "ami-123",
                  name: "builder",
                  description: "Build image",
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
          ],
        },
      }),
    );

    expect(Object.keys(details)).toEqual([
      "Instance ID",
      "Image ID",
      "Name",
      "Description",
      "State",
      "Creation Date",
      "Architecture",
      "Image Type",
      "Root Device Type",
      "Root Device Name",
      "Virtualization Type",
      "Hypervisor",
      "Owner ID",
    ]);
    expect(details).toEqual({
      "Instance ID": "i-123",
      "Image ID": "ami-123",
      Name: "builder",
      Description: "Build image",
      State: "available",
      "Creation Date": "2026-05-21T12:00:00Z",
      Architecture: "x86_64",
      "Image Type": "machine",
      "Root Device Type": "ebs",
      "Root Device Name": "/dev/sda1",
      "Virtualization Type": "hvm",
      Hypervisor: "xen",
      "Owner ID": "111122223333",
    });
  });

  it("uses image metadata when the output is empty", () => {
    const details = createImageMapper.getExecutionDetails(
      buildDetailsContext({
        metadata: { imageId: "ami-empty", state: "pending" },
      }),
    );

    expect(details).toEqual({
      "Image ID": "ami-empty",
      State: "pending",
    });
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Create image",
    componentName: "aws.ec2.createImage",
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
