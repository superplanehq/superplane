import { describe, expect, it } from "vitest";
import { getEventRunTitle, getEventTitleFallback } from "./utils";

describe("event title helpers", () => {
  it("falls back to the received-at timestamp when runTitle is empty", () => {
    const createdAt = "2026-04-21T12:34:56Z";

    expect(
      getEventRunTitle({
        id: "event-1",
        createdAt,
        data: {},
        nodeId: "node-1",
        type: "test.event",
      }),
    ).toBe(getEventTitleFallback(createdAt));
  });

  it("prefers an explicit fallback over the timestamp fallback", () => {
    expect(
      getEventRunTitle(
        {
          id: "event-1",
          createdAt: "2026-04-21T12:34:56Z",
          data: {},
          nodeId: "node-1",
          type: "test.event",
        },
        "Run",
      ),
    ).toBe("Run");
  });

  it("returns the trimmed run title when present", () => {
    expect(
      getEventRunTitle({
        id: "event-1",
        createdAt: "2026-04-21T12:34:56Z",
        runTitle: "  Push to main  ",
        data: {},
        nodeId: "node-1",
        type: "test.event",
      }),
    ).toBe("Push to main");
  });
});
