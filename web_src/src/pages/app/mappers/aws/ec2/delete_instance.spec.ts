import { describe, expect, it } from "bun:test";
import { createEC2Fixtures } from "./test-support";

import { deleteInstanceMapper } from "./delete_instance";

const { buildOutput, buildDetailsCtx, buildComponentCtx } = createEC2Fixtures({
  nodeName: "Delete Instance",
  componentName: "ec2.deleteInstance",
  label: "EC2 • Delete Instance",
  icon: "server",
});

describe("deleteInstanceMapper.props", () => {
  it("includes instance name and region in metadata", () => {
    const props = deleteInstanceMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          instance: "i-abc123",
        },
        metadata: {
          instanceName: "my-instance",
          region: "us-east-1",
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "my-instance" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });

  it("falls back to instance id when name metadata is missing", () => {
    const props = deleteInstanceMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          instance: "i-abc123",
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "i-abc123" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });
});

describe("deleteInstanceMapper.getExecutionDetails", () => {
  it("shows deleted at and region while termination is in progress", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "us-east-1",
          instance: "i-abc123",
        },
      },
      execution: {
        outputs: undefined,
      },
    });

    const details = deleteInstanceMapper.getExecutionDetails(ctx);
    expect(details["Deleted At"]).toBe(new Date("2026-05-21T12:01:00Z").toLocaleString());
    expect(details["Region"]).toBe("us-east-1");
    expect(details["State"]).toBe("-");
    expect(details["Instance ID"]).toBeUndefined();
  });

  it("maps terminated output fields without instance id", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "us-east-1",
        },
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              instance: "i-abc123",
              state: "terminated",
            }),
          ],
        },
      },
    });

    const details = deleteInstanceMapper.getExecutionDetails(ctx);
    expect(details["Deleted At"]).toBe(new Date("2026-05-21T12:01:00Z").toLocaleString());
    expect(details["State"]).toBe("terminated");
    expect(details["Instance ID"]).toBeUndefined();
  });
});
