import { describe, expect, it } from "bun:test";

import { getTunnelMapper } from "./get_tunnel";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createNodeFixtures } from "./test_support";

const { buildDetailsCtx } = createNodeFixtures("cloudflare.getTunnel");
const buildOutput = (data: unknown) => buildCloudflareOutput("cloudflare.tunnel.fetched", data);

const tunnelOutputData = {
  tunnel: {
    id: "tun123",
    name: "edge-tunnel",
    status: "healthy",
    config_src: "cloudflare",
  },
  accountId: "acc123",
};

describe("getTunnelMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getTunnelMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("includes tunnel fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput(tunnelOutputData)] },
      },
    });
    const details = getTunnelMapper.getExecutionDetails(ctx);
    expect(details["Tunnel ID"]).toBeUndefined();
    expect(details["Name"]).toBe("edge-tunnel");
    expect(details["Status"]).toBe("healthy");
  });
});

describe("eventStateRegistry.getTunnel", () => {
  it("maps finished passed to fetched", () => {
    expect(eventStateRegistry.getTunnel.getState(buildExecution())).toBe("fetched");
  });
});
