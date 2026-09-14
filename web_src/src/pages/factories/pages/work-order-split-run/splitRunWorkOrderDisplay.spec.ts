import { describe, expect, it } from "bun:test";

import { formatDuration, formatMinutesSecondsDuration } from "@/lib/duration";

import {
  displayStatusForLineStatus,
  durationForExecution,
  elapsedForDisplay,
  implementationRunnerModel,
} from "./splitRunWorkOrderDisplay";

const START = "2026-08-21T12:00:00.000Z";
const FOUR_MINUTES = 4 * 60 * 1000;

describe("displayStatusForLineStatus", () => {
  it("maps line status to the card icon status", () => {
    expect(displayStatusForLineStatus("running")).toBe("running");
    expect(displayStatusForLineStatus("passed")).toBe("completed");
    expect(displayStatusForLineStatus("waiting")).toBe("waiting");
    expect(displayStatusForLineStatus("failed")).toBe("failed");
    expect(displayStatusForLineStatus("cancelled")).toBe("cancelled");
    expect(displayStatusForLineStatus("pending")).toBe("draft");
  });
});

describe("elapsedForDisplay", () => {
  it("keeps draft and waiting labels", () => {
    expect(elapsedForDisplay("draft")).toBe("Not started");
    expect(elapsedForDisplay("waiting", { createdAt: START, updatedAt: START })).toBe("Waiting");
  });

  it("formats running elapsed time from createdAt", () => {
    const now = Date.parse(START) + FOUR_MINUTES;
    expect(elapsedForDisplay("running", { createdAt: START, updatedAt: START }, now)).toBe(
      `${formatDuration(FOUR_MINUTES, { precision: "second" })} so far`,
    );
  });

  it("formats completed elapsed time from createdAt to updatedAt", () => {
    const updatedAt = new Date(Date.parse(START) + FOUR_MINUTES).toISOString();
    expect(elapsedForDisplay("completed", { createdAt: START, updatedAt })).toBe(
      formatDuration(FOUR_MINUTES, { precision: "second" }),
    );
  });
});

describe("implementationRunnerModel", () => {
  it("returns empty when no implement phase has a model", () => {
    expect(implementationRunnerModel([{ id: "backlog", name: "Backlog", status: "passed" }])).toBe("");
    expect(implementationRunnerModel([{ id: "implement-0", name: "Implement", status: "running" }])).toBe("");
  });

  it("fills Auto from the implementation canvas", () => {
    expect(
      implementationRunnerModel(
        [{ id: "implement-0", name: "Implement", status: "running" }],
        [{ configuration: { model: "hosted::openrouter::x-ai/grok-4.6" } }],
      ),
    ).toBe("grok-4.6");
  });

  it("uses the executed runner when the canvas has unused runners", () => {
    expect(
      implementationRunnerModel(
        [{ id: "implement-0", name: "Implement", status: "running" }],
        [
          { id: "ran", configuration: { model: "hosted::openrouter::x-ai/grok-4.6" } },
          { id: "idle", configuration: { model: "anthropic/claude-sonnet-4-6" } },
        ],
        { ran: "running", idle: "did_not_run" },
      ),
    ).toBe("grok-4.6");
  });

  it("shortens the active implement model", () => {
    expect(
      implementationRunnerModel([
        { id: "backlog", name: "Backlog", status: "passed", model: "opus" },
        {
          id: "implement-0",
          name: "Implement",
          status: "running",
          canvasKey: "implementation",
          model: "hosted::openrouter::x-ai/grok-4.6",
        },
      ]),
    ).toBe("grok-4.6");
  });

  it("prefers the active implement phase over an older one", () => {
    expect(
      implementationRunnerModel([
        { id: "implement-0", name: "Implement", status: "passed", model: "anthropic/claude-sonnet-4-6" },
        { id: "implement-1", name: "Implement", status: "running", model: "hosted::openrouter::x-ai/grok-4.6" },
      ]),
    ).toBe("grok-4.6");
  });

  it("uses the latest implement phase when none is active", () => {
    expect(
      implementationRunnerModel([
        { id: "implement-0", name: "Implement", status: "passed", model: "anthropic/claude-sonnet-4-6" },
        { id: "verify-1", name: "Verify", status: "passed", model: "ignored" },
        { id: "implement-1", name: "Implement", status: "passed", model: "hosted::openrouter::x-ai/grok-4.6" },
      ]),
    ).toBe("grok-4.6");
  });
});

describe("durationForExecution", () => {
  it("formats a finished step from createdAt to updatedAt", () => {
    const updatedAt = new Date(Date.parse(START) + FOUR_MINUTES).toISOString();
    expect(durationForExecution({ createdAt: START, updatedAt }, "passed")).toBe(
      formatMinutesSecondsDuration(FOUR_MINUTES),
    );
  });

  it("formats a running step against now", () => {
    const now = Date.parse(START) + FOUR_MINUTES;
    expect(durationForExecution({ createdAt: START, updatedAt: START }, "running", now)).toBe(
      formatMinutesSecondsDuration(FOUR_MINUTES),
    );
  });

  it("shows a short duration when a finished step has no elapsed time", () => {
    expect(durationForExecution({ createdAt: START, updatedAt: START }, "passed")).toBe("<1s");
  });
});
