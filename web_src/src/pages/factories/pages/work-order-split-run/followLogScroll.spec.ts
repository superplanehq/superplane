import { describe, expect, it } from "bun:test";

import {
  FOLLOW_BOTTOM_THRESHOLD_PX,
  FOLLOW_RESUME_THRESHOLD_PX,
  followAfterRunningPhaseChange,
  isNearLogBottom,
  nextFollowAfterScroll,
  showJumpToLatest,
} from "./followLogScroll";

describe("followAfterRunningPhaseChange", () => {
  it("turns Follow on when a phase starts running", () => {
    expect(followAfterRunningPhaseChange(false, null, "implement")).toBe(true);
  });

  it("turns Follow on when a later phase starts running", () => {
    expect(followAfterRunningPhaseChange(false, "implement", "verify")).toBe(true);
  });

  it("keeps Follow as the user left it when the run finishes", () => {
    expect(followAfterRunningPhaseChange(true, "implement", null)).toBe(true);
    expect(followAfterRunningPhaseChange(false, "implement", null)).toBe(false);
  });

  it("does not turn Follow back on while the same phase stays running", () => {
    expect(followAfterRunningPhaseChange(false, "implement", "implement")).toBe(false);
  });
});

describe("isNearLogBottom", () => {
  it("is true within 144 pixels of the bottom", () => {
    expect(isNearLogBottom(56, 300, 100)).toBe(true);
  });

  it("is false when the user scrolls more than 144 pixels up", () => {
    expect(isNearLogBottom(55, 300, 100)).toBe(false);
  });
});

describe("nextFollowAfterScroll", () => {
  it("stops follow on an upward scroll even inside the leave band", () => {
    expect(
      nextFollowAfterScroll({
        following: true,
        resumeOnBottom: true,
        distanceFromBottom: 80,
        scrollingUp: true,
      }),
    ).toBe(false);
  });

  it("does not resume until the user scrolls down to the bottom", () => {
    expect(
      nextFollowAfterScroll({
        following: false,
        resumeOnBottom: true,
        distanceFromBottom: 80,
        scrollingUp: false,
      }),
    ).toBe(false);
    expect(
      nextFollowAfterScroll({
        following: false,
        resumeOnBottom: true,
        distanceFromBottom: FOLLOW_RESUME_THRESHOLD_PX,
        scrollingUp: false,
      }),
    ).toBe(true);
  });
});

describe("showJumpToLatest", () => {
  it("stays hidden while the user is still inside the leave band", () => {
    expect(showJumpToLatest(false, FOLLOW_BOTTOM_THRESHOLD_PX)).toBe(false);
    expect(showJumpToLatest(false, FOLLOW_BOTTOM_THRESHOLD_PX + 1)).toBe(true);
  });
});
