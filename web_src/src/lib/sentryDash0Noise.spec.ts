import { describe, expect, it } from "vitest";
import type { ErrorEvent, EventHint } from "@sentry/react";
import { dash0TelemetryNoisePatterns, isDash0TelemetryNoiseEvent } from "@/lib/sentryDash0Noise";

describe("dash0TelemetryNoisePatterns", () => {
  it("matches Dash0 SDK telemetry-failure console messages", () => {
    const dash0FailureMessages = [
      "Failed to send telemetry to https://ingress.us-west-2.aws.dash0.com/v1/logs: 400 Bad Request",
      "Error sending telemetry to https://ingress.us-west-2.aws.dash0.com/v1/traces: TypeError: NetworkError",
      "Unable to send telemetry, fetch is not defined",
      "Failed to transmit logs Error: timed out",
      "Failed to transmit spans Error: timed out",
    ];

    for (const message of dash0FailureMessages) {
      expect(
        dash0TelemetryNoisePatterns.some((pattern) => pattern.test(message)),
        `expected "${message}" to be ignored`,
      ).toBe(true);
    }
  });

  it("does not match unrelated application messages", () => {
    const realApplicationMessages = [
      "Failed to load canvas: HTTP 500",
      "Unhandled promise rejection in /signup form",
      "TypeError: Cannot read properties of undefined (reading 'foo')",
    ];

    for (const message of realApplicationMessages) {
      expect(
        dash0TelemetryNoisePatterns.some((pattern) => pattern.test(message)),
        `expected "${message}" NOT to be ignored`,
      ).toBe(false);
    }
  });
});

describe("isDash0TelemetryNoiseEvent", () => {
  it("drops console-captured TypeError: Failed to fetch from Dash0 transport errors", () => {
    const event = {
      exception: {
        values: [
          {
            type: "TypeError",
            value: "Failed to fetch",
            mechanism: { type: "console", handled: false },
          },
        ],
      },
      extra: {
        arguments: ["Failed to transmit logs", new TypeError("Failed to fetch")],
      },
    } satisfies ErrorEvent;

    expect(isDash0TelemetryNoiseEvent(event)).toBe(true);
  });

  it("drops console-captured span transmission failures", () => {
    const event = {
      exception: {
        values: [
          {
            type: "TypeError",
            value: "Failed to fetch",
            mechanism: { type: "console", handled: false },
          },
        ],
      },
      extra: {
        arguments: ["Failed to transmit spans", new TypeError("Failed to fetch")],
      },
    } satisfies ErrorEvent;

    expect(isDash0TelemetryNoiseEvent(event)).toBe(true);
  });

  it("does not drop unrelated console-captured fetch failures", () => {
    const event = {
      exception: {
        values: [
          {
            type: "TypeError",
            value: "Failed to fetch",
            mechanism: { type: "console", handled: false },
          },
        ],
      },
      extra: {
        arguments: ["Failed to save canvas", new TypeError("Failed to fetch")],
      },
    } satisfies ErrorEvent;

    expect(isDash0TelemetryNoiseEvent(event)).toBe(false);
  });

  it("does not drop unhandled fetch failures without Dash0 console context", () => {
    const event = {
      exception: {
        values: [
          {
            type: "TypeError",
            value: "Failed to fetch",
            mechanism: { type: "onunhandledrejection", handled: false },
          },
        ],
      },
    } satisfies ErrorEvent;

    expect(isDash0TelemetryNoiseEvent(event)).toBe(false);
  });

  it("reads console arguments from the event hint when extra is missing", () => {
    const event = {
      exception: {
        values: [
          {
            type: "TypeError",
            value: "Failed to fetch",
            mechanism: { type: "console", handled: false },
          },
        ],
      },
    } satisfies ErrorEvent;
    const hint = {
      captureContext: {
        extra: {
          arguments: ["Failed to transmit logs", new TypeError("Failed to fetch")],
        },
      },
    } satisfies EventHint;

    expect(isDash0TelemetryNoiseEvent(event, hint)).toBe(true);
  });
});
