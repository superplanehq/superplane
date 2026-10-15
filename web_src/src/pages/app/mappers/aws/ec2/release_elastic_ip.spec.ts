import { describe, expect, it } from "bun:test";
import { createEC2Fixtures } from "./test-support";

import { releaseElasticIPMapper } from "./release_elastic_ip";

const { buildDetailsCtx, buildComponentCtx } = createEC2Fixtures({
  nodeName: "Release Elastic IP",
  componentName: "ec2.releaseElasticIP",
  label: "EC2 • Release Elastic IP",
  icon: "server",
});

describe("releaseElasticIPMapper.props", () => {
  it("shows allocation ID and region in metadata", () => {
    const props = releaseElasticIPMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", allocationId: "eipalloc-abc123" },
        metadata: { region: "us-east-1", allocationId: "eipalloc-abc123" },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "hash", label: "eipalloc-abc123" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });
});

describe("releaseElasticIPMapper.getExecutionDetails", () => {
  it("shows region while output is pending", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1", allocationId: "eipalloc-abc123" },
      },
      execution: { outputs: undefined },
    });

    const details = releaseElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Completed At"]).toBeTruthy();
    expect(details["Region"]).toBe("us-east-1");
  });

  it("prefers output region over configuration", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "eu-west-1", allocationId: "eipalloc-abc123" } },
      execution: {
        outputs: {
          default: [
            {
              type: "aws.ec2.elastic-ip.released",
              timestamp: new Date().toISOString(),
              data: {
                allocationId: "eipalloc-abc123",
                region: "us-east-1",
              },
            },
          ],
        },
      },
    });

    const details = releaseElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Region"]).toBe("us-east-1");
    expect(details["Allocation ID"]).toBeUndefined();
  });
});
