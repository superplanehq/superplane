import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { getKVValueMapper } from "./get_kv_value";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.getKVValue",
  label: "Get KV Value",
  description: "",
  icon: "cloud",
  color: "orange",
};

const { buildNode, buildDetailsCtx, buildPropsContext } = createComponentFixtures(defaultDefinition);
const buildOutput = (data: unknown) => buildCloudflareOutput("cloudflare.kv.value.fetched", data);

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("getKVValueMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getKVValueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getKVValueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts namespace id, key, and value from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ accountId: "acc123", namespaceId: "ns123", key: "my-key", value: "my-value" })],
        },
      },
    });
    const details = getKVValueMapper.getExecutionDetails(ctx);
    expect(details["Namespace ID"]).toBeUndefined();
    expect(details["Key"]).toBe("my-key");
    expect(details["Value"]).toBe("my-value");
  });

  it("uses dash placeholders when result fields are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = getKVValueMapper.getExecutionDetails(ctx);
    expect(details["Namespace ID"]).toBeUndefined();
    expect(details["Key"]).toBe("-");
    expect(details["Value"]).toBe("-");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ namespaceId: "ns123", key: "my-key", value: "val" })] },
      },
    });
    expect(getKVValueMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("getKVValueMapper.props", () => {
  it("shows namespaceName and keyName from node metadata", () => {
    const props = getKVValueMapper.props(
      buildPropsContext({
        node: buildNode({
          metadata: { namespaceName: "My Namespace", keyName: "my-key" },
          configuration: { namespace: "ns123", kvKey: "my-key" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "database", label: "My Namespace" },
      { icon: "key", label: "my-key" },
    ]);
  });

  it("falls back to config ids when metadata names are absent", () => {
    const props = getKVValueMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { namespace: "ns123", kvKey: "my-key" } }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "database", label: "ns123" },
      { icon: "key", label: "my-key" },
    ]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = getKVValueMapper.props(buildPropsContext({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.getKVValue", () => {
  it("maps finished success to fetched", () => {
    expect(eventStateRegistry.getKVValue.getState(buildExecution())).toBe("fetched");
  });
});
