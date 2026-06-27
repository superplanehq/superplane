import type { ErrorEvent, EventHint } from "@sentry/react";
import { describe, expect, it } from "vitest";
import { filterNonActionableErrors } from "@/sentry";

function makeReferenceErrorEvent(message: string): ErrorEvent {
  return {
    exception: {
      values: [
        {
          type: "ReferenceError",
          value: message,
        },
      ],
    },
  } as ErrorEvent;
}

describe("filterNonActionableErrors", () => {
  it("drops Safari ReferenceError events for short minified identifiers", () => {
    const event = makeReferenceErrorEvent("Can't find variable: oU");

    expect(filterNonActionableErrors(event, {})).toBeNull();
  });

  it("drops V8 ReferenceError events for short minified identifiers", () => {
    const event = makeReferenceErrorEvent("oU is not defined");

    expect(filterNonActionableErrors(event, {})).toBeNull();
  });

  it("drops when only the originalException hint identifies the error", () => {
    const event = { exception: { values: [{ type: "ReferenceError" }] } } as ErrorEvent;
    const hint: EventHint = {
      originalException: new ReferenceError("Can't find variable: oU"),
    };

    expect(filterNonActionableErrors(event, hint)).toBeNull();
  });

  it("keeps ReferenceErrors that reference descriptive identifier names", () => {
    const event = makeReferenceErrorEvent("Can't find variable: someUnboundFunction");

    expect(filterNonActionableErrors(event, {})).toBe(event);
  });

  it("keeps unrelated errors with short identifier-like values", () => {
    const event = {
      exception: {
        values: [
          {
            type: "TypeError",
            value: "oU is not defined",
          },
        ],
      },
    } as ErrorEvent;

    expect(filterNonActionableErrors(event, {})).toBe(event);
  });

  it("keeps events that have no exception payload", () => {
    const event = {} as ErrorEvent;

    expect(filterNonActionableErrors(event, {})).toBe(event);
  });
});
