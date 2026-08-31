import { describe, expect, it } from "vitest";
import type { CanvasesCanvas } from "@/api-client";

import type { CanvasSpecNode } from "./columnCanvasAgent";
import { deriveFactoryAppResetWiring, resolveFactoryAppTemplate } from "./factoryAppTemplate";

function canvasWith(nodes: CanvasSpecNode[]): CanvasesCanvas {
  return { metadata: { id: "app-1" }, spec: { nodes, edges: [] } };
}

describe("resolveFactoryAppTemplate", () => {
  it("matches the Plan app by its entrypoint node id", () => {
    const canvas = canvasWith([{ id: "onrun-create-plan", component: "onRun" }]);
    expect(resolveFactoryAppTemplate(canvas)?.id).toBe("line-planning");
  });

  it("matches the Implement app by its entrypoint node id", () => {
    const canvas = canvasWith([{ id: "onrun-implement", component: "onRun" }]);
    expect(resolveFactoryAppTemplate(canvas)?.id).toBe("line-implementation");
  });

  it("matches the PR Closure app by its entrypoint node id", () => {
    const canvas = canvasWith([{ id: "on-pr-closed", component: "github.onPullRequest" }]);
    expect(resolveFactoryAppTemplate(canvas)?.id).toBe("pr-closure");
  });

  it("returns null for a canvas with no matching bundled template", () => {
    const canvas = canvasWith([{ id: "some-other-node", component: "onRun" }]);
    expect(resolveFactoryAppTemplate(canvas)).toBeNull();
  });

  it("returns null for the software-factory canvas", () => {
    // The software-factory template's own entrypoint node id must never match.
    const canvas = canvasWith([{ id: "onrun-create-plan", component: "onRun" }]);
    expect(resolveFactoryAppTemplate(canvas)?.id).not.toBe("software-factory");
  });

  it("returns null when the canvas has no nodes", () => {
    expect(resolveFactoryAppTemplate(canvasWith([]))).toBeNull();
  });

  it("returns null when the canvas is missing", () => {
    expect(resolveFactoryAppTemplate(null)).toBeNull();
    expect(resolveFactoryAppTemplate(undefined)).toBeNull();
  });
});

describe("deriveFactoryAppResetWiring", () => {
  it("reads appRepository and defaultBranch from a literal repository/base field", () => {
    const canvas = canvasWith([
      {
        id: "create-pr",
        component: "github.createPullRequest",
        configuration: { repository: "acme/web", base: "develop" },
        integration: { id: "int-1", name: "github-acme" },
      },
    ]);

    const wiring = deriveFactoryAppResetWiring(canvas);
    expect(wiring.installParams).toEqual({ appRepository: "acme/web", defaultBranch: "develop" });
    expect(wiring.integrations).toEqual({ github: { id: "int-1", name: "github-acme", ready: true } });
  });

  it("falls back to the agent runner's REPO/BASE environment values", () => {
    const canvas = canvasWith([
      {
        id: "planner-agent-no-issue",
        component: "runnerClaudeCode",
        configuration: {
          model: "opus",
          credentials: { source: "hosted" },
          environment: [{ name: "REPO", value: "acme/api" }],
        },
      },
    ]);

    const wiring = deriveFactoryAppResetWiring(canvas);
    expect(wiring.installParams).toEqual({ appRepository: "acme/api" });
    expect(wiring.agentRewrite).toEqual({
      component: "runnerClaudeCode",
      model: "opus",
      credentials: { source: "hosted" },
    });
  });

  it("carries the live agent node's integration credentials", () => {
    const canvas = canvasWith([
      {
        id: "implementation-agent-no-issue",
        component: "runnerCodex",
        configuration: {
          model: "gpt-5",
          credentials: { source: "integration", integration: { name: "codex-acme" } },
        },
      },
    ]);

    const wiring = deriveFactoryAppResetWiring(canvas);
    expect(wiring.agentRewrite).toEqual({
      component: "runnerCodex",
      model: "gpt-5",
      credentials: { source: "integration", name: "codex-acme" },
    });
  });

  it("omits install params, integrations, and agent rewrite when nothing is found", () => {
    const canvas = canvasWith([{ id: "on-pr-closed", component: "github.onPullRequest", configuration: {} }]);

    const wiring = deriveFactoryAppResetWiring(canvas);
    expect(wiring.installParams).toEqual({});
    expect(wiring.integrations).toEqual({});
    expect(wiring.agentRewrite).toBeUndefined();
  });

  it("handles a missing canvas", () => {
    expect(deriveFactoryAppResetWiring(null)).toEqual({ installParams: {}, integrations: {}, agentRewrite: undefined });
  });
});
