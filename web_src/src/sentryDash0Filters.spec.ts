import type { Breadcrumb, ErrorEvent } from "@sentry/react";
import { describe, expect, it } from "vitest";
import {
  isDash0TelemetryConsoleMessage,
  shouldDropDash0TelemetryBreadcrumb,
  shouldDropDash0TelemetryEvent,
} from "@/sentryDash0Filters";

describe("sentryDash0Filters", () => {
  describe("isDash0TelemetryConsoleMessage", () => {
    it("matches Dash0 export failure console messages", () => {
      expect(
        isDash0TelemetryConsoleMessage(
          "Error sending telemetry to https://ingress.us-west-2.aws.dash0.com/v1/traces: TypeError: Failed to fetch",
        ),
      ).toBe(true);
      expect(
        isDash0TelemetryConsoleMessage(
          "Failed to send telemetry to https://ingress.us-west-2.aws.dash0.com/v1/traces: 400 Bad Request",
        ),
      ).toBe(true);
    });

    it("does not match unrelated console messages", () => {
      expect(isDash0TelemetryConsoleMessage("Failed to fetch canvas data")).toBe(false);
      expect(isDash0TelemetryConsoleMessage("TypeError: Failed to fetch")).toBe(false);
    });
  });

  describe("shouldDropDash0TelemetryBreadcrumb", () => {
    it("drops Dash0 telemetry console breadcrumbs", () => {
      const breadcrumb: Breadcrumb = {
        category: "console",
        message:
          "Error sending telemetry to https://ingress.us-west-2.aws.dash0.com/v1/traces: TypeError: Failed to fetch",
      };

      expect(shouldDropDash0TelemetryBreadcrumb(breadcrumb)).toBe(true);
    });

    it("keeps unrelated console breadcrumbs", () => {
      const breadcrumb: Breadcrumb = {
        category: "console",
        message: "Canvas load failed",
      };

      expect(shouldDropDash0TelemetryBreadcrumb(breadcrumb)).toBe(false);
    });
  });

  describe("shouldDropDash0TelemetryEvent", () => {
    it("drops Failed to fetch events paired with Dash0 telemetry console breadcrumbs", () => {
      const event: ErrorEvent = {
        type: undefined,
        exception: {
          values: [{ type: "TypeError", value: "Failed to fetch" }],
        },
        breadcrumbs: [
          {
            category: "console",
            message:
              "Error sending telemetry to https://ingress.us-west-2.aws.dash0.com/v1/traces: TypeError: Failed to fetch",
          },
        ],
      };

      expect(shouldDropDash0TelemetryEvent(event)).toBe(true);
    });

    it("drops events whose message is a Dash0 telemetry console warning", () => {
      const event: ErrorEvent = {
        type: undefined,
        exception: {
          values: [
            {
              type: "Error",
              value: "Failed to send telemetry to https://ingress.us-west-2.aws.dash0.com/v1/traces: 400 Bad Request",
            },
          ],
        },
      };

      expect(shouldDropDash0TelemetryEvent(event)).toBe(true);
    });

    it("keeps unrelated Failed to fetch events", () => {
      const event: ErrorEvent = {
        type: undefined,
        exception: {
          values: [{ type: "TypeError", value: "Failed to fetch" }],
        },
        breadcrumbs: [
          {
            category: "fetch",
            message: "GET https://app.superplane.com/api/v1/canvases",
          },
        ],
      };

      expect(shouldDropDash0TelemetryEvent(event)).toBe(false);
    });
  });
});
