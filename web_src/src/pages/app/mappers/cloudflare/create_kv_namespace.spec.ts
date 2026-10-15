import { describe, expect, it } from "bun:test";

import type { ComponentDefinition, ExecutionInfo } from "../types";
import { createKVNamespaceMapper } from "./create_kv_namespace";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.createKVNamespace",
  label: "Create KV Namespace",
  description: "",
  icon: "cloud",
  color: "orange",
};

const { buildNode, buildDetailsCtx, buildPropsContext } = createComponentFixtures(defaultDefinition);
const buildOutput = (data: unknown) => buildCloudflareOutput("cloudflare.kv.namespace.created", data);

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("createKVNamespaceMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createKVNamespaceMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createKVNamespaceMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data has no namespace key", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => createKVNamespaceMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts namespace fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              accountId: "acc123",
              namespace: { id: "ns123", title: "my-namespace" },
            }),
          ],
        },
      },
    });
    const details = createKVNamespaceMapper.getExecutionDetails(ctx);
    expect(details["Title"]).toBe("my-namespace");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ namespace: { id: "ns123", title: "my-namespace" } })],
        },
      },
    });
    expect(createKVNamespaceMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("createKVNamespaceMapper.props", () => {
  it("includes namespace title from configuration in metadata", () => {
    const props = createKVNamespaceMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { title: "my-namespace" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "database", label: "my-namespace" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = createKVNamespaceMapper.props(buildPropsContext({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.createKVNamespace", () => {
  it("maps finished success to created", () => {
    expect(eventStateRegistry.createKVNamespace.getState(buildExecution())).toBe("created");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.createKVNamespace.getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.createKVNamespace.getState(failed)).toBe("failed");
  });
});
