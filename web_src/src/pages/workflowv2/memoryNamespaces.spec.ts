import { describe, expect, it } from "vitest";

import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { countMemoryNamespaces } from "./memoryNamespaces";

function entry(namespace: string): CanvasMemoryEntry {
  return { id: `${namespace}-${Math.random()}`, namespace, values: {} };
}

describe("countMemoryNamespaces", () => {
  it("returns 0 for an empty entry list", () => {
    expect(countMemoryNamespaces([])).toBe(0);
  });

  it("counts a single namespace once regardless of entry count", () => {
    expect(countMemoryNamespaces([entry("envs"), entry("envs"), entry("envs")])).toBe(1);
  });

  it("counts distinct namespaces", () => {
    expect(countMemoryNamespaces([entry("envs"), entry("machines"), entry("envs")])).toBe(2);
  });

  it("collapses empty and whitespace namespaces into a single bucket", () => {
    expect(countMemoryNamespaces([entry(""), entry("   "), entry("envs")])).toBe(2);
  });
});
