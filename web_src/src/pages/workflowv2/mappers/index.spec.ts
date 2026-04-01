import { describe, expect, it } from "vitest";
import type { CanvasesCanvasNodeExecution, ComponentsNode } from "@/api-client";
import { getExecutionDetails } from "./index";

function makeNode(name: string): ComponentsNode {
  return {
    id: "node-1",
    component: { name },
  } as ComponentsNode;
}

function makeExecution(): CanvasesCanvasNodeExecution {
  return {
    id: "execution-1",
    createdAt: new Date().toISOString(),
  } as CanvasesCanvasNodeExecution;
}

describe("getExecutionDetails", () => {
  it("returns undefined when no top-level mapper is registered", () => {
    expect(getExecutionDetails("unknown-component", makeExecution(), makeNode("unknown-component"))).toBeUndefined();
  });

  it("returns undefined when no app mapper is registered", () => {
    expect(getExecutionDetails("unknown.component", makeExecution(), makeNode("unknown.component"))).toBeUndefined();
  });

  it("returns undefined when an app exists but the component mapper does not", () => {
    expect(getExecutionDetails("github.unknown", makeExecution(), makeNode("github.unknown"))).toBeUndefined();
  });
});
