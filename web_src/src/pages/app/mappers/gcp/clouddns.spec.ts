import { describe, expect, it } from "bun:test";

import { cloudDNSMapper } from "./clouddns";
import { buildComponentCtx, buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

const COMPLETED_AT = "2026-06-08T09:01:00Z";

describe("cloudDNSMapper.props", () => {
  it("surfaces managed zone, record name, and type", () => {
    const props = cloudDNSMapper.props(
      buildComponentCtx(
        { configuration: { managedZone: "example-zone", name: "api.example.com", type: "A" } },
        "gcp.clouddns.createRecord",
      ),
    );

    expect(props.metadata).toEqual([
      { icon: "globe", label: "example-zone" },
      { icon: "tag", label: "api.example.com" },
      { icon: "layers", label: "A" },
    ]);
  });
});

describe("cloudDNSMapper.getExecutionDetails", () => {
  it("maps change and record fields from the output", () => {
    const details = cloudDNSMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [
              buildOutput(
                {
                  change: { id: "change-1", status: "done" },
                  record: { name: "api.example.com.", type: "A" },
                },
                "gcp.clouddns.change",
              ),
            ],
          },
        },
      }),
    );

    expect(details["Change ID"]).toBe("change-1");
    expect(details["Status"]).toBe("done");
    expect(details["Record Name"]).toBe("api.example.com.");
    expect(details["Record Type"]).toBe("A");
  });

  it("includes completed at from the payload timestamp", () => {
    const details = cloudDNSMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [
              {
                type: "gcp.clouddns.change",
                timestamp: COMPLETED_AT,
                data: { change: { id: "change-2" } },
              },
            ],
          },
        },
      }),
    );

    expect(details["Completed At"]).toBe(new Date(COMPLETED_AT).toLocaleString());
    expect(details["Change ID"]).toBe("change-2");
  });

  it("does not throw when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => cloudDNSMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(cloudDNSMapper.getExecutionDetails(ctx)).toEqual({});
  });
});
