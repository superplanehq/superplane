import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { deletePoolMapper } from "./delete_pool";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.deletePool",
  label: "Delete Pool",
  description: "",
  icon: "cloud",
  color: "orange",
};

const { buildNode, buildDetailsCtx, buildPropsContext } = createComponentFixtures(defaultDefinition);
const buildOutput = (data: unknown) => buildCloudflareOutput("cloudflare.pool", data);

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("deletePoolMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deletePoolMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deletePoolMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts pool id and deleted status from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ poolId: "pool123", accountId: "acc123", deleted: true })],
        },
      },
    });
    const details = deletePoolMapper.getExecutionDetails(ctx);
    expect(details["Pool ID"]).toBe("pool123");
    expect(details["Deleted"]).toBe("true");
  });

  it("uses dash placeholders when result fields are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = deletePoolMapper.getExecutionDetails(ctx);
    expect(details["Pool ID"]).toBe("-");
    expect(details["Deleted"]).toBe("-");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ poolId: "pool123", deleted: true })] },
      },
    });
    expect(deletePoolMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("deletePoolMapper.props", () => {
  it("prefers pool name from node metadata", () => {
    const props = deletePoolMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { pool: "pool-id" },
          metadata: { poolName: "My Pool" },
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "My Pool" }]);
  });

  it("prefers pool name from snake_case node metadata keys", () => {
    const props = deletePoolMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { pool: "pool-id" },
          metadata: { pool_name: "My Pool" },
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "My Pool" }]);
  });

  it("falls back to pool id from configuration when metadata is absent", () => {
    const props = deletePoolMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { pool: "pool-id" },
          metadata: {},
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "pool-id" }]);
  });

  it("returns empty metadata when both metadata and configuration are empty", () => {
    const props = deletePoolMapper.props(buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.deletePool", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry.deletePool.getState(buildExecution())).toBe("deleted");
  });
});
