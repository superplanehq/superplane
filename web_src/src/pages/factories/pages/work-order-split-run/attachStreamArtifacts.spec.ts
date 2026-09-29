import { describe, expect, it } from "bun:test";

import type { FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

import { attachStreamArtifacts, phasesWithRunArtifacts, streamArtifactIndexFromEvents } from "./attachStreamArtifacts";
import type { SplitRunPhase, SplitRunStreamLine } from "./splitRunMocks";

const NOTE: FactoriesWorkOrderArtifact = {
  id: "art-md-1",
  type: "TYPE_MARKDOWN",
  data: { title: "PLAN.md", body: "Ship the retry fix." },
};

const UPDATED_NOTE: FactoriesWorkOrderArtifact = {
  ...NOTE,
  data: { ...NOTE.data, title: "PLAN.md", body: "Ship the retry fix. Updated." },
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
  run?: { id?: string },
): FactoriesWorkOrderEvent {
  return {
    type: "order.artifact.added",
    timestamp: at,
    event: {
      ...(automation ? { automation } : {}),
      ...(run ? { run } : {}),
      artifact,
    },
  };
}

describe("attachStreamArtifacts", () => {
  it("hangs the matching artifact on a stream line by event nodeId", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-output"), streamLine("noop")],
      [
        artifactAddedEvent("2026-08-24T16:32:18.000Z", NOTE, { nodeId: "add-output" }),
        { type: "order.comment.added", timestamp: "2026-08-24T16:33:00.000Z", event: {} },
      ],
    );

    expect(stream?.[0]?.artifact).toEqual(NOTE);
    expect(stream?.[1]?.artifact).toBeUndefined();
  });

  it("overlays live artifact data on the event snapshot", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-output")],
      [
        artifactAddedEvent(
          "2026-08-24T16:32:18.000Z",
          { id: "art-md-1", type: "markdown", data: { title: "PLAN.md", body: "Ship the retry fix." } },
          { nodeId: "add-output" },
        ),
      ],
      [UPDATED_NOTE],
    );

    expect(stream?.[0]?.artifact).toEqual(UPDATED_NOTE);
  });

  it("keeps the later artifact when the same node adds twice", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-output")],
      [
        artifactAddedEvent("2026-08-24T16:32:18.000Z", NOTE, { nodeId: "add-output" }),
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
        artifactAddedEvent("2026-08-24T16:32:18.000Z", NOTE, { nodeId: "add-output" }),
      ],
    );

    expect(stream?.[0]?.artifact).toEqual(BRANCH);
  });

  it("leaves the stream unchanged when the event has no node", () => {
    const lines = [streamLine("noop")];
    const stream = attachStreamArtifacts(lines, [artifactAddedEvent("2026-08-24T16:32:18.000Z", NOTE)]);

    expect(stream?.[0]?.artifact).toBeUndefined();
    expect(stream?.[0]).toEqual(lines[0]);
  });

  it("leaves the stream unchanged when no line matches the nodeId", () => {
    const stream = attachStreamArtifacts(
      [streamLine("noop")],
      [artifactAddedEvent("2026-08-24T16:32:18.000Z", NOTE, { nodeId: "add-output" })],
    );

    expect(stream?.[0]?.artifact).toBeUndefined();
  });

  it("matches nodeName to the line component name when nodeId is missing", () => {
    const stream = attachStreamArtifacts(
      [streamLine("node-2", "noop 2")],
      [artifactAddedEvent("2026-08-24T16:35:18.000Z", NOTE, { nodeName: "noop 2" })],
    );

    expect(stream?.[0]?.artifact).toEqual(NOTE);
  });

  it("does not use nodeName when the event already has a nodeId", () => {
    const stream = attachStreamArtifacts(
      [streamLine("other-node", "noop 2")],
      [artifactAddedEvent("2026-08-24T16:35:18.000Z", NOTE, { nodeId: "add-output", nodeName: "noop 2" })],
    );

    expect(stream?.[0]?.artifact).toBeUndefined();
  });

  it("returns undefined when the stream is missing", () => {
    expect(attachStreamArtifacts(undefined, [])).toBeUndefined();
  });

  it("hangs a pull request event on a stream line by nodeId", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-pr")],
      [
        {
          type: "order.pull_request.added",
          timestamp: "2026-08-24T16:32:18.000Z",
          event: {
            automation: { nodeId: "add-pr" },
            pullRequest: {
              id: "pr-1",
              number: 482,
              url: "https://github.com/example/ledger/pull/482",
              state: "open",
            },
          },
        },
      ],
    );

    expect(stream?.[0]?.artifact).toBeUndefined();
    expect(stream?.[0]?.pullRequest).toMatchObject({
      id: "pr-1",
      number: "482",
      url: "https://github.com/example/ledger/pull/482",
      state: "STATE_OPEN",
    });
  });

  it("does not attach an artifact produced by a different run when scoped by nodeId", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-output")],
      [artifactAddedEvent("2026-08-24T16:32:18.000Z", NOTE, { nodeId: "add-output" }, { id: "run-a" })],
      undefined,
      undefined,
      "run-b",
    );

    expect(stream?.[0]?.artifact).toBeUndefined();
  });

  it("attaches an artifact produced by the matching run when scoped by nodeId", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-output")],
      [artifactAddedEvent("2026-08-24T16:32:18.000Z", NOTE, { nodeId: "add-output" }, { id: "run-a" })],
      undefined,
      undefined,
      "run-a",
    );

    expect(stream?.[0]?.artifact).toEqual(NOTE);
  });

  it("does not attach an artifact produced by a different run when scoped by nodeName", () => {
    const stream = attachStreamArtifacts(
      [streamLine("node-2", "Add Task Artifact")],
      [artifactAddedEvent("2026-08-24T16:35:18.000Z", NOTE, { nodeName: "Add Task Artifact" }, { id: "run-planning" })],
      undefined,
      undefined,
      "run-pr-feedback",
    );

    expect(stream?.[0]?.artifact).toBeUndefined();
  });

  it("attaches an artifact with no run reference regardless of the requested run", () => {
    const stream = attachStreamArtifacts(
      [streamLine("add-output")],
      [artifactAddedEvent("2026-08-24T16:32:18.000Z", NOTE, { nodeId: "add-output" })],
      undefined,
      undefined,
      "run-b",
    );

    expect(stream?.[0]?.artifact).toEqual(NOTE);
  });
});

function stage(runId: string | undefined): SplitRunPhase {
  return {
    id: "phase-1",
    name: "Analysis",
    status: "passed",
    duration: "1s",
    componentName: "Analysis",
    artifacts: [],
    stream: [],
    canvasSteps: [],
    runId,
  };
}

describe("phasesWithRunArtifacts", () => {
  it("keeps every artifact a run produced on that stage", () => {
    const index = streamArtifactIndexFromEvents(
      [
        artifactAddedEvent("2026-08-24T16:32:18.000Z", NOTE, { nodeId: "write-spec" }, { id: "run-analysis" }),
        artifactAddedEvent("2026-08-24T16:38:18.000Z", BRANCH, { nodeId: "write-spec" }, { id: "run-analysis" }),
        artifactAddedEvent(
          "2026-08-24T16:40:18.000Z",
          { id: "art-other", type: "TYPE_LINK", data: { url: "https://example.com" } },
          { nodeId: "other" },
          { id: "run-other" },
        ),
      ],
      undefined,
    );

    const phases = phasesWithRunArtifacts([stage("run-analysis"), stage("run-other")], index);

    expect(phases[0]?.artifacts.map((artifact) => artifact.id)).toEqual(["art-md-1", "art-branch-1"]);
    expect(phases[1]?.artifacts.map((artifact) => artifact.id)).toEqual(["art-other"]);
  });

  it("attaches a plan spec by canvas run id when the event has no run", () => {
    const spec: FactoriesWorkOrderArtifact = {
      id: "art-spec",
      type: "TYPE_MARKDOWN",
      data: { name: "spec.md", title: "spec.md", body: "# Plan", canvasRunId: "run-analysis" },
    };
    const index = streamArtifactIndexFromEvents([], [spec]);

    const phases = phasesWithRunArtifacts([stage("run-analysis"), stage("run-implement")], index);

    expect(phases[0]?.artifacts).toEqual([spec]);
    expect(phases[1]?.artifacts).toEqual([]);
  });

  it("attaches a broadcast to the node that posted it", () => {
    const stream = attachStreamArtifacts(
      [streamLine("preview", "Create preview")],
      [
        {
          type: "order.activity.broadcast",
          timestamp: "2026-08-24T16:34:00.000Z",
          event: {
            title: "Preview environment ready",
            url: "https://preview.example.com/pr/42",
            automation: { nodeId: "preview" },
          },
        },
      ],
    );

    expect(stream?.[0]?.broadcasts).toEqual([
      { title: "Preview environment ready", body: undefined, url: "https://preview.example.com/pr/42" },
    ]);
  });
});
