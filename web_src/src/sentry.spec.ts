import { describe, expect, it } from "vitest";
import type { ErrorEvent, EventHint } from "@sentry/react";
import { filterNonActionableErrors } from "./sentry";

function buildEvent(overrides: Partial<ErrorEvent> = {}): ErrorEvent {
  return {
    exception: {
      values: [],
    },
    ...overrides,
  } as ErrorEvent;
}

describe("filterNonActionableErrors", () => {
  it("drops Safari ReferenceErrors about minified identifiers", () => {
    const event = buildEvent({
      exception: {
        values: [
          {
            type: "ReferenceError",
            value: "Can't find variable: Gy",
          },
        ],
      },
    });

    expect(filterNonActionableErrors(event, {})).toBeNull();
  });

  it("drops V8 ReferenceErrors about minified identifiers", () => {
    const event = buildEvent({
      exception: {
        values: [
          {
            type: "ReferenceError",
            value: "vl is not defined",
          },
        ],
      },
    });

    expect(filterNonActionableErrors(event, {})).toBeNull();
  });

  it("drops events whose original exception is a minified ReferenceError", () => {
    const event = buildEvent();
    const hint: EventHint = {
      originalException: new ReferenceError("Can't find variable: Gy"),
    };

    expect(filterNonActionableErrors(event, hint)).toBeNull();
  });

  it("keeps ReferenceErrors that mention a real source identifier", () => {
    const event = buildEvent({
      exception: {
        values: [
          {
            type: "ReferenceError",
            value: "posthog is not defined",
          },
        ],
      },
    });

    expect(filterNonActionableErrors(event, {})).toBe(event);
  });

  it("keeps non-ReferenceError events even when their value looks short", () => {
    const event = buildEvent({
      exception: {
        values: [
          {
            type: "TypeError",
            value: "Gy is not defined",
          },
        ],
      },
    });

    expect(filterNonActionableErrors(event, {})).toBe(event);
  });

  it("keeps unrelated events", () => {
    const event = buildEvent({
      exception: {
        values: [
          {
            type: "TypeError",
            value: "Cannot read properties of undefined (reading 'foo')",
          },
        ],
      },
    });

    expect(filterNonActionableErrors(event, {})).toBe(event);
  });
});
