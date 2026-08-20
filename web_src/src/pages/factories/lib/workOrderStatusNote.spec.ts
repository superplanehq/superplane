import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrderStatusNote } from "@/api-client";

import { presentWorkOrderStatusNotes } from "./workOrderStatusNote";

function note(overrides: Partial<FactoriesWorkOrderStatusNote> = {}): FactoriesWorkOrderStatusNote {
  return {
    key: "pr-closure",
    kind: "info",
    headline: "Review the pull request",
    ...overrides,
  };
}

describe("presentWorkOrderStatusNotes", () => {
  it("keeps an unflagged note while a line is running", () => {
    const presented = presentWorkOrderStatusNotes([note()], "running");

    expect(presented.map((entry) => entry.key)).toEqual(["pr-closure"]);
  });

  it("hides a waiting-only note while a line is running", () => {
    const presented = presentWorkOrderStatusNotes([note({ showOnlyWhenWaiting: true })], "running");

    expect(presented).toEqual([]);
  });

  it("shows a waiting-only note while the order is waiting", () => {
    const presented = presentWorkOrderStatusNotes([note({ showOnlyWhenWaiting: true })], "waiting");

    expect(presented.map((entry) => entry.key)).toEqual(["pr-closure"]);
  });

  it("keeps unflagged notes and drops waiting-only notes while running", () => {
    const presented = presentWorkOrderStatusNotes(
      [
        note({ key: "pr-closure" }),
        note({ key: "queue-slot", headline: "Waiting for a slot", showOnlyWhenWaiting: true }),
      ],
      "running",
    );

    expect(presented.map((entry) => entry.key)).toEqual(["pr-closure"]);
  });
});
