import * as Sentry from "@sentry/react";

interface SentryWindow extends Window {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
}

// Dash0 Web SDK logs export failures to the console; captureConsoleIntegration would forward them.
const DASH0_TELEMETRY_CONSOLE_IGNORE = /^(Failed to send telemetry to|Error sending telemetry to|Failed to fetch)/;

// Browser extensions (e.g. Vue Devtools) emit console warnings that captureConsoleIntegration
// would otherwise surface as application errors even though they do not originate from our code.
const BROWSER_EXTENSION_CONSOLE_IGNORE = /^Another version of Vue Devtools/;

// Third-party console noise that should never be reported to Sentry.
const CONSOLE_IGNORE_PATTERNS = [DASH0_TELEMETRY_CONSOLE_IGNORE, BROWSER_EXTENSION_CONSOLE_IGNORE];

export function isIgnoredConsoleMessage(message: string): boolean {
  return CONSOLE_IGNORE_PATTERNS.some((pattern) => pattern.test(message));
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
    ignoreErrors: CONSOLE_IGNORE_PATTERNS,
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
