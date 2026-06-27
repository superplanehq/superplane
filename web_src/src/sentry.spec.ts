import type { ErrorEvent, EventHint } from "@sentry/react";
import { describe, expect, it } from "vitest";
import { filterNonActionableErrors } from "@/sentry";

function buildEvent(overrides: Partial<ErrorEvent> = {}): ErrorEvent {
  return {
    type: undefined,
    ...overrides,
  } as ErrorEvent;
}

function referenceErrorEvent(value: string): ErrorEvent {
  return buildEvent({
    exception: {
      values: [{ type: "ReferenceError", value }],
    },
  });
}

describe("filterNonActionableErrors", () => {
  it("drops Safari ReferenceErrors with short minified identifiers", () => {
    const event = referenceErrorEvent("Can't find variable: Z");

    expect(filterNonActionableErrors(event, {})).toBeNull();
  });

  it("drops V8 / Firefox ReferenceErrors with short minified identifiers", () => {
    const event = referenceErrorEvent("oU is not defined");

    expect(filterNonActionableErrors(event, {})).toBeNull();
  });

  it("drops events whose originalException hint is a minified ReferenceError", () => {
    const hint: EventHint = { originalException: new ReferenceError("Can't find variable: vl") };
    const event = buildEvent();

    expect(filterNonActionableErrors(event, hint)).toBeNull();
  });

  it("keeps ReferenceErrors that reference descriptive identifiers", () => {
    const event = referenceErrorEvent("useExecutionState is not defined");

    expect(filterNonActionableErrors(event, {})).toBe(event);
  });

  it("keeps non-ReferenceError exceptions even when the value is short", () => {
    const event = buildEvent({
      exception: {
        values: [{ type: "TypeError", value: "Z is not defined" }],
      },
    });

    expect(filterNonActionableErrors(event, {})).toBe(event);
  });

  it("keeps events without exception payloads", () => {
    const event = buildEvent();

    expect(filterNonActionableErrors(event, {})).toBe(event);
  });

  it("ignores hint originalException when it is not a ReferenceError", () => {
    const hint: EventHint = { originalException: new TypeError("Can't find variable: Z") };
    const event = buildEvent();

    expect(filterNonActionableErrors(event, hint)).toBe(event);
  });
});
