import * as Sentry from "@sentry/react";

interface SentryWindow extends Window {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
}

// Dash0 Web SDK logs export failures to the console; captureConsoleIntegration would forward them.
const DASH0_TELEMETRY_CONSOLE_IGNORE = /^(Failed to send telemetry to|Error sending telemetry to)/;

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
    beforeBreadcrumb(breadcrumb) {
      if (
        breadcrumb.category === "console" &&
        typeof breadcrumb.message === "string" &&
        DASH0_TELEMETRY_CONSOLE_IGNORE.test(breadcrumb.message)
      ) {
        return null;
      }

      return breadcrumb;
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
