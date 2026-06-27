import { describe, expect, it } from "vitest";
import type { ErrorEvent, EventHint } from "@sentry/react";
import { filterBrowserNetworkErrors } from "@/sentry";

function makeHint(originalException: EventHint["originalException"] = undefined): EventHint {
  return { originalException };
}

function makeEvent(values: { type?: string; value?: string }[] = []): ErrorEvent {
  return {
    exception: {
      values: values.map((v) => ({
        type: v.type,
        value: v.value,
      })),
    },
  } as ErrorEvent;
}

describe("filterBrowserNetworkErrors", () => {
  it("drops events whose original exception is a browser network error", () => {
    expect(filterBrowserNetworkErrors(makeEvent(), makeHint(new TypeError("Failed to fetch")))).toBeNull();
    expect(filterBrowserNetworkErrors(makeEvent(), makeHint(new TypeError("Load failed")))).toBeNull();
    expect(filterBrowserNetworkErrors(makeEvent(), makeHint(new Error("Load failed")))).toBeNull();
    expect(
      filterBrowserNetworkErrors(makeEvent(), makeHint(new Error("NetworkError when attempting to fetch resource."))),
    ).toBeNull();
  });

  it("drops events whose serialized exception value is a browser network error", () => {
    expect(
      filterBrowserNetworkErrors(makeEvent([{ type: "TypeError", value: "Failed to fetch" }]), makeHint()),
    ).toBeNull();
    expect(filterBrowserNetworkErrors(makeEvent([{ type: "TypeError", value: "Load failed" }]), makeHint())).toBeNull();
  });

  it("keeps events for actionable errors", () => {
    const error = new Error("Something went wrong");
    const event = makeEvent([{ type: "Error", value: "Something went wrong" }]);
    expect(filterBrowserNetworkErrors(event, makeHint(error))).toBe(event);
  });

  it("keeps events when there is no exception information at all", () => {
    const event = { exception: undefined } as unknown as ErrorEvent;
    expect(filterBrowserNetworkErrors(event, makeHint())).toBe(event);
  });
});
