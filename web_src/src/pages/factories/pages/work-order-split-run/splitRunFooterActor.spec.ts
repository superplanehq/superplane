import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrderEvent } from "@/api-client";
import { createOrgUserDisplayLookup, type OrgUserDisplay } from "@/lib/orgUserDisplay";

import { footerCloserFromEvents } from "./splitRunFooterActor";

const ALEX: OrgUserDisplay = {
  id: "user-alex",
  name: "Alex Rivera",
  initials: "AR",
  avatarUrl: "https://example.com/alex.png",
};

const resolveUser = createOrgUserDisplayLookup(new Map([[ALEX.id, ALEX]]));

function statusEvent(
  at: string,
  payload: {
    userId?: string;
    automationName?: string;
    toState?: string;
    toResult?: string;
  },
): FactoriesWorkOrderEvent {
  return {
    type: "order.status.updated",
    timestamp: at,
    event: {
      ...(payload.userId ? { user: { id: payload.userId } } : {}),
      ...(payload.automationName ? { automation: { appName: payload.automationName } } : {}),
      toState: payload.toState ?? "closed",
      ...(payload.toResult !== undefined ? { toResult: payload.toResult } : {}),
    },
  };
}

function stepFinishedEvent(at: string, payload: { userId?: string; runResult?: string }): FactoriesWorkOrderEvent {
  return {
    type: "step.execution.finished",
    timestamp: at,
    event: {
      ...(payload.userId ? { user: { id: payload.userId } } : {}),
      run: { id: "run-1", result: payload.runResult ?? "cancelled" },
    },
  };
}

describe("footerCloserFromEvents", () => {
  it("names the person who completed the work order, including avatar", () => {
    expect(
      footerCloserFromEvents(
        [statusEvent("2026-08-04T12:00:00.000Z", { userId: ALEX.id, toResult: "completed" })],
        "completed",
        resolveUser,
      ),
    ).toEqual({ actor: ALEX });
  });

  it("names the person who rejected the work order", () => {
    expect(
      footerCloserFromEvents(
        [statusEvent("2026-08-04T12:00:00.000Z", { userId: ALEX.id, toResult: "rejected" })],
        "rejected",
        resolveUser,
      ),
    ).toEqual({ actor: ALEX });
  });

  it("names the automation that closed the work order when no person is set", () => {
    expect(
      footerCloserFromEvents(
        [statusEvent("2026-08-04T12:00:00.000Z", { automationName: "PR Closure", toResult: "completed" })],
        "completed",
        resolveUser,
      ),
    ).toEqual({ automationName: "PR Closure" });
  });

  it("uses the latest matching close, not an older one", () => {
    expect(
      footerCloserFromEvents(
        [
          statusEvent("2026-08-04T10:00:00.000Z", { userId: "user-old", toResult: "completed" }),
          statusEvent("2026-08-04T12:00:00.000Z", { userId: ALEX.id, toResult: "completed" }),
        ],
        "completed",
        resolveUser,
      ).actor?.id,
    ).toBe(ALEX.id);
  });

  it("names the person who stopped the automation from a cancelled step finish", () => {
    expect(
      footerCloserFromEvents(
        [stepFinishedEvent("2026-08-04T12:00:00.000Z", { userId: ALEX.id })],
        "waiting",
        resolveUser,
      ),
    ).toEqual({ actor: ALEX });
  });

  it("falls back to a cancelled status event when no step finish is present", () => {
    expect(
      footerCloserFromEvents(
        [statusEvent("2026-08-04T12:00:00.000Z", { userId: ALEX.id, toState: "open", toResult: "cancelled" })],
        "waiting",
        resolveUser,
      ),
    ).toEqual({ actor: ALEX });
  });

  it("prefers a cancelled step finish over an older status event", () => {
    expect(
      footerCloserFromEvents(
        [
          statusEvent("2026-08-04T10:00:00.000Z", { userId: "user-old", toState: "open", toResult: "cancelled" }),
          stepFinishedEvent("2026-08-04T12:00:00.000Z", { userId: ALEX.id }),
        ],
        "running",
        resolveUser,
      ).actor?.id,
    ).toBe(ALEX.id);
  });

  it("returns empty when no matching event exists, so the footer keeps A person", () => {
    expect(footerCloserFromEvents([], "completed", resolveUser)).toEqual({});
    expect(
      footerCloserFromEvents(
        [statusEvent("2026-08-04T12:00:00.000Z", { userId: ALEX.id, toState: "open", toResult: "" })],
        "waiting",
        resolveUser,
      ),
    ).toEqual({});
  });

  it("ignores a completed close when the footer is for a rejected order", () => {
    expect(
      footerCloserFromEvents(
        [statusEvent("2026-08-04T12:00:00.000Z", { userId: ALEX.id, toResult: "completed" })],
        "rejected",
        resolveUser,
      ),
    ).toEqual({});
  });
});
