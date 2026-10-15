import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { putKVValueMapper } from "./put_kv_value";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.putKVValue",
  label: "Put KV Value",
  description: "",
  icon: "cloud",
  color: "orange",
};

const { buildNode, buildDetailsCtx, buildPropsContext } = createComponentFixtures(defaultDefinition);
const buildOutput = (data: unknown) => buildCloudflareOutput("cloudflare.kv.value.written", data);

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("putKVValueMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => putKVValueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => putKVValueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ accountId: "acc123", namespaceId: "ns123", key: "my-key", written: true })],
        },
      },
    });
    const details = putKVValueMapper.getExecutionDetails(ctx);
    expect(details["Namespace ID"]).toBeUndefined();
    expect(details["Key"]).toBe("my-key");
    expect(details["Written"]).toBe("true");
  });

  it("uses dash placeholders when result fields are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = putKVValueMapper.getExecutionDetails(ctx);
    expect(details["Namespace ID"]).toBeUndefined();
    expect(details["Key"]).toBe("-");
    expect(details["Written"]).toBe("-");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ namespaceId: "ns123", key: "my-key", written: true })] },
      },
    });
    expect(putKVValueMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("putKVValueMapper.props", () => {
  it("shows namespaceName and keyName from node metadata", () => {
    const props = putKVValueMapper.props(
      buildPropsContext({
        node: buildNode({
          metadata: { namespaceName: "My Namespace", keyName: "my-key" },
          configuration: { namespace: "ns123", key: "my-key" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "database", label: "My Namespace" },
      { icon: "key", label: "my-key" },
    ]);
  });

  it("falls back to config ids when metadata names are absent", () => {
    const props = putKVValueMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { namespace: "ns123", key: "my-key" } }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "database", label: "ns123" },
      { icon: "key", label: "my-key" },
    ]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = putKVValueMapper.props(buildPropsContext({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.putKVValue", () => {
  it("maps finished success to success", () => {
    expect(eventStateRegistry.putKVValue.getState(buildExecution())).toBe("success");
  });
});
