import { describe, expect, it } from "bun:test";
import { createEC2Fixtures } from "./test-support";

import { allocateElasticIPMapper } from "./allocate_elastic_ip";

const { buildDetailsCtx, buildComponentCtx } = createEC2Fixtures({
  nodeName: "Allocate Elastic IP",
  componentName: "ec2.allocateElasticIP",
  label: "EC2 • Allocate Elastic IP",
  icon: "server",
});

describe("allocateElasticIPMapper.props", () => {
  it("shows region in metadata for Amazon pool", () => {
    const props = allocateElasticIPMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", ipSource: "amazon" },
        metadata: { region: "us-east-1", ipSource: "amazon" },
      }),
    );

    expect(props.metadata).toEqual([{ icon: "globe", label: "us-east-1" }]);
  });

  it("shows IP source and region for BYOIP pool", () => {
    const props = allocateElasticIPMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", ipSource: "byoip", publicIpv4Pool: "ipv4pool-ec2-abc123" },
        metadata: { region: "us-east-1", ipSource: "byoip" },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "layers", label: "BYOIP pool" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });
});

describe("allocateElasticIPMapper.getExecutionDetails", () => {
  it("shows placeholder values while output is not yet available", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "us-east-1" } },
      execution: { outputs: undefined },
    });

    const details = allocateElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Completed At"]).toBeTruthy();
    expect(details["Region"]).toBe("us-east-1");
    expect(details["Public IP"]).toBe("-");
  });

  it("maps public IP and region from output", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "us-east-1" } },
      execution: {
        outputs: {
          default: [
            {
              type: "aws.ec2.elastic-ip.allocated",
              timestamp: new Date().toISOString(),
              data: {
                allocationId: "eipalloc-abc123",
                publicIp: "203.0.113.10",
                region: "us-east-1",
              },
            },
          ],
        },
      },
    });

    const details = allocateElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Public IP"]).toBe("203.0.113.10");
    expect(details["Region"]).toBe("us-east-1");
  });
});
