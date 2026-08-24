import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

import { attachStreamArtifacts } from "./attachStreamArtifacts";
import type { SplitRunStreamLine } from "./splitRunMocks";

const OPEN_PR: FactoriesWorkOrderArtifact = {
  id: "art-pr-1",
  type: "TYPE_PR",
  data: { number: 482, state: "open", url: "https://github.com/example/ledger/pull/482" },
};

const MERGED_PR: FactoriesWorkOrderArtifact = {
  ...OPEN_PR,
  data: { ...OPEN_PR.data, state: "merged" },
};

const BRANCH: FactoriesWorkOrderArtifact = {
  id: "art-branch-1",
  type: "TYPE_BRANCH",
  data: { name: "feature/refund-retry" },
};

function streamLine(nodeId: string, componentName = nodeId): SplitRunStreamLine {
  return {
    id: nodeId,
    nodeId,
    at: "16:32:18",
    componentName,
    status: "passed",
  };
}

function artifactAddedEvent(
  at: string,
  artifact: { id?: string; type?: string; data?: Record<string, unknown> },
  automation?: { nodeId?: string; nodeName?: string },
): FactoriesWorkOrderEvent {
  return {
    type: "order.artifact.added",
    timestamp: at,
    event: {
      ...(automation ? { automation } : {}),
      artifact,
    },
  };
}

describe("attachStreamArtifacts", () => {
  it("hangs the matching artifact on a stream line by event nodeId", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-pr"), streamLine("noop")],
      [
        artifactAddedEvent("2026-08-24T16:32:18.000Z", OPEN_PR, { nodeId: "add-pr" }),
        { type: "order.comment.added", timestamp: "2026-08-24T16:33:00.000Z", event: {} },
      ],
    );

    expect(stream?.[0]?.artifact).toEqual(OPEN_PR);
    expect(stream?.[1]?.artifact).toBeUndefined();
  });

  it("overlays live artifact data on the event snapshot", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-pr")],
      [
        artifactAddedEvent(
          "2026-08-24T16:32:18.000Z",
          { id: "art-pr-1", type: "pr", data: { number: 482, state: "open" } },
          { nodeId: "add-pr" },
        ),
      ],
      [MERGED_PR],
    );

    expect(stream?.[0]?.artifact).toEqual(MERGED_PR);
  });

  it("keeps the later artifact when the same node adds twice", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-output")],
      [
        artifactAddedEvent("2026-08-24T16:32:18.000Z", OPEN_PR, { nodeId: "add-output" }),
        artifactAddedEvent("2026-08-24T16:38:18.000Z", BRANCH, { nodeId: "add-output" }),
      ],
    );

    expect(stream?.[0]?.artifact).toEqual(BRANCH);
  });

  it("sorts newest-first pages so the later timestamp still wins", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-output")],
      [
        artifactAddedEvent("2026-08-24T16:38:18.000Z", BRANCH, { nodeId: "add-output" }),
        artifactAddedEvent("2026-08-24T16:32:18.000Z", OPEN_PR, { nodeId: "add-output" }),
      ],
    );

    expect(stream?.[0]?.artifact).toEqual(BRANCH);
  });

  it("leaves the stream unchanged when the event has no node", () => {
    const lines = [streamLine("noop")];
    const stream = attachStreamArtifacts(lines, [artifactAddedEvent("2026-08-24T16:32:18.000Z", OPEN_PR)]);

    expect(stream?.[0]?.artifact).toBeUndefined();
    expect(stream?.[0]).toEqual(lines[0]);
  });

  it("leaves the stream unchanged when no line matches the nodeId", () => {
    const stream = attachStreamArtifacts(
      [streamLine("noop")],
      [artifactAddedEvent("2026-08-24T16:32:18.000Z", OPEN_PR, { nodeId: "add-pr" })],
    );

    expect(stream?.[0]?.artifact).toBeUndefined();
  });

  it("matches nodeName to the line component name when nodeId is missing", () => {
    const stream = attachStreamArtifacts(
      [streamLine("node-2", "noop 2")],
      [artifactAddedEvent("2026-08-24T16:35:18.000Z", OPEN_PR, { nodeName: "noop 2" })],
    );

    expect(stream?.[0]?.artifact).toEqual(OPEN_PR);
  });

  it("does not use nodeName when the event already has a nodeId", () => {
    const stream = attachStreamArtifacts(
      [streamLine("other-node", "noop 2")],
      [artifactAddedEvent("2026-08-24T16:35:18.000Z", OPEN_PR, { nodeId: "add-pr", nodeName: "noop 2" })],
    );

    expect(stream?.[0]?.artifact).toBeUndefined();
  });

  it("returns undefined when the stream is missing", () => {
    expect(attachStreamArtifacts(undefined, [])).toBeUndefined();
  });
});
