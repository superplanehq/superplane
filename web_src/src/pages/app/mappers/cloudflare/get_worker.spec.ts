import { describe, expect, it } from "bun:test";

import type { ComponentBaseContext } from "../types";
import { getWorkerMapper } from "./get_worker";
import { eventStateRegistry } from "./index";
import { buildExecution, createNodeFixtures } from "./test_support";

const { buildDetailsCtx } = createNodeFixtures("cloudflare.getWorker");

describe("getWorkerMapper.getExecutionDetails", () => {
  it("includes deployment count", () => {
    const data = {
      workerScript: "w",
      deployments: [{ id: "d1" }, { id: "d2" }],
      settings: { compatibility_date: "2024-01-01" },
    };
    const outputs = { default: [{ type: "cloudflare.worker.fetched", timestamp: new Date().toISOString(), data }] };
    const details = getWorkerMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs } }));
    expect(details["Script"]).toBe("w");
    expect(details["Deployments"]).toBe("2");
    expect(details["Compatibility date"]).toBe("2024-01-01");
  });
});

describe("getWorkerMapper.props metadata", () => {
  it("prefers resolved script display name from node metadata", () => {
    const props = getWorkerMapper.props({
      nodes: [],
      node: {
        id: "n1",
        name: "Node",
        componentName: "cloudflare.getWorker",
        isCollapsed: false,
        configuration: { workerScript: "id-1" },
        metadata: { scriptDisplayName: "My Worker" },
      },
      componentDefinition: {
        name: "cloudflare.getWorker",
        label: "Get Worker",
        description: "",
        icon: "cloud",
        color: "orange",
      },
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
    } satisfies ComponentBaseContext);
    expect(props.metadata).toEqual([{ icon: "code", label: "My Worker" }]);
  });
});

describe("eventStateRegistry.getWorker", () => {
  it("maps finished success to fetched", () => {
    expect(eventStateRegistry.getWorker.getState(buildExecution())).toBe("fetched");
  });
});
