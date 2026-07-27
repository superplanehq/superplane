import yaml from "js-yaml";
import { describe, expect, it } from "vitest";

import canvasYamlRaw from "./canvas.yaml?raw";

type Node = {
  id?: string;
  component?: string;
  configuration?: { node?: string; parameters?: { CheckName?: string } };
};
type Edge = { sourceId?: string; targetId?: string; channel?: string };
type Canvas = { spec?: { nodes?: Node[]; edges?: Edge[] } };

const canvas = yaml.load(canvasYamlRaw) as Canvas;
const nodes = canvas.spec?.nodes ?? [];
const edges = canvas.spec?.edges ?? [];

// The "Update Progress" app (onrun-onrun-2nvcpr) checks a box in the PR body by
// reading the whole body, flipping one line, and writing it back. Any two of
// these invocations that run concurrently race and clobber each other's box.
const UPDATE_PROGRESS_NODE = "onrun-onrun-2nvcpr";

const progressWriterIds = new Set(
  nodes
    .filter((node) => node.component === "runApp" && node.configuration?.node === UPDATE_PROGRESS_NODE)
    .map((node) => node.id)
    .filter((id): id is string => Boolean(id)),
);

describe("software-factory canvas", () => {
  it("has more than one PR-progress checkbox writer to guard", () => {
    // Sanity check: the invariant below is meaningless if the template only has
    // one writer. If the factory is restructured this keeps the test honest.
    expect(progressWriterIds.size).toBeGreaterThan(1);
  });

  it("never fans out two PR-progress checkbox writers from the same trigger", () => {
    const bySourceChannel = new Map<string, string[]>();
    for (const edge of edges) {
      if (!edge.targetId || !progressWriterIds.has(edge.targetId)) continue;
      const key = `${edge.sourceId ?? ""}::${edge.channel ?? ""}`;
      bySourceChannel.set(key, [...(bySourceChannel.get(key) ?? []), edge.targetId]);
    }

    const parallel = [...bySourceChannel.entries()].filter(([, targets]) => targets.length > 1);
    expect(parallel).toEqual([]);
  });

  it("checks off 'Human review requested' after 'Confirmed CI passes'", () => {
    // The two survivors of the race are chained: CI-passes is marked first, then
    // human-review-requested reads the already-updated body and adds its box.
    const ciWriter = nodes.find((n) => n.configuration?.parameters?.CheckName === "Confirmed CI passes");
    const reviewWriter = nodes.find((n) => n.configuration?.parameters?.CheckName === "Human review requested");
    expect(ciWriter?.id).toBeTruthy();
    expect(reviewWriter?.id).toBeTruthy();

    const chained = edges.some(
      (edge) => edge.sourceId === ciWriter?.id && edge.targetId === reviewWriter?.id && edge.channel === "passed",
    );
    expect(chained).toBe(true);
  });
});
