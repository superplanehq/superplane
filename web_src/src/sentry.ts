import * as Sentry from "@sentry/react";

interface SentryWindow extends Window {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
}

// captureConsoleIntegration forwards every console.warn/error to Sentry, including noise that
// does not originate from application code. These patterns drop known third-party sources:
//   - Dash0 Web SDK logs telemetry export failures to the console.
//   - Browser extensions (e.g. Vue Devtools) log conflicts detected in the user's browser.
const IGNORED_CONSOLE_MESSAGES = [
  /^Failed to send telemetry to/,
  /^Error sending telemetry to/,
  /^Failed to fetch/,
  /^Another version of Vue Devtools/,
];

export function isIgnoredConsoleMessage(message: string): boolean {
  return IGNORED_CONSOLE_MESSAGES.some((pattern) => pattern.test(message));
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
    ignoreErrors: IGNORED_CONSOLE_MESSAGES,
    beforeSend(event) {
      const frames = event.exception?.values?.[0]?.stacktrace?.frames ?? [];
      const allDash0 = frames.length > 0 && frames.every((frame) => frame.filename?.includes("@dash0/sdk-web"));
      if (allDash0) {
        return null;
      }

      return event;
    },
    beforeBreadcrumb(breadcrumb) {
      if (
        breadcrumb.category === "console" &&
        typeof breadcrumb.message === "string" &&
        isIgnoredConsoleMessage(breadcrumb.message)
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
