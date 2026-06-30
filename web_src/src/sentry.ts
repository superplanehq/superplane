import * as Sentry from "@sentry/react";
import {
  DASH0_TELEMETRY_CONSOLE_IGNORE,
  shouldDropDash0TelemetryBreadcrumb,
  shouldDropDash0TelemetryEvent,
} from "@/sentryDash0Filters";

interface SentryWindow extends Window {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
}

let dsn: string | undefined;
let environment: string | undefined;

if (typeof window !== "undefined") {
  const sentryWindow = window as SentryWindow;
  dsn = sentryWindow.SUPERPLANE_SENTRY_DSN;
  environment = sentryWindow.SUPERPLANE_SENTRY_ENVIRONMENT;
}

if (dsn) {
  Sentry.init({
    dsn,
    environment,
    ignoreErrors: [DASH0_TELEMETRY_CONSOLE_IGNORE],
    beforeSend(event) {
      return shouldDropDash0TelemetryEvent(event) ? null : event;
    },
    beforeBreadcrumb(breadcrumb) {
      return shouldDropDash0TelemetryBreadcrumb(breadcrumb) ? null : breadcrumb;
    },
    integrations: [
      Sentry.captureConsoleIntegration({
        levels: ["warn", "error"],
      }),
      Sentry.browserApiErrorsIntegration({
        setTimeout: true,
        setInterval: true,
        requestAnimationFrame: true,
        XMLHttpRequest: true,
        eventTarget: true,
      }),
      Sentry.globalHandlersIntegration({
        onerror: true,
        onunhandledrejection: true,
      }),
    ],
  });
}

export { Sentry };
