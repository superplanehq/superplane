import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { deployWorkerMapper } from "./deploy_worker";
import { eventStateRegistry } from "./index";
import { buildExecution, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.deployWorker",
  label: "Deploy Worker",
  description: "",
  icon: "cloud",
  color: "orange",
};

const { buildNode, buildDetailsCtx, buildPropsContext } = createComponentFixtures(defaultDefinition);

describe("deployWorkerMapper.getExecutionDetails", () => {
  it("includes script only when data is present", () => {
    const data = {
      scriptName: "my-worker",
      versionId: "ver-1",
      deployment: { id: "dep-1" },
    };
    const outputs = { default: [{ type: "cloudflare.worker.deployed", timestamp: new Date().toISOString(), data }] };
    const details = deployWorkerMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs } }));
    expect(details["Script"]).toBe("my-worker");
    expect(details["Version ID"]).toBeUndefined();
    expect(details["Deployment ID"]).toBeUndefined();
  });
});

describe("deployWorkerMapper.props", () => {
  it("shows at most three chips: name, source, provision", () => {
    const props = deployWorkerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { scriptName: "w", source: "inline" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "code", label: "w" },
      { icon: "file-text", label: "Inline" },
      { icon: "package", label: "Provision on" },
    ]);
  });

  it("shows provision off when disabled", () => {
    const props = deployWorkerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { scriptName: "w", source: "inline", provisionIfMissing: false },
        }),
      }),
    );
    expect(props.metadata?.[2]).toEqual({ icon: "package", label: "Provision off" });
  });
});

describe("eventStateRegistry.deployWorker", () => {
  it("maps deploy output to deployed", () => {
    const execution = buildExecution({
      outputs: {
        default: [
          {
            type: "cloudflare.worker.deployed",
            timestamp: new Date().toISOString(),
            data: {},
          },
        ],
      },
    });

    expect(eventStateRegistry.deployWorker.getState(execution)).toBe("deployed");
  });
});
