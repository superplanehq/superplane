import * as Sentry from "@sentry/react";

interface SentryWindow extends Window {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
}

// Console messages emitted by third parties (our telemetry SDK, browser
// extensions) that captureConsoleIntegration would otherwise forward to Sentry
// as application errors. These are not bugs in our code, so we drop them.
export const IGNORED_CONSOLE_MESSAGES = [
  // Dash0 Web SDK logs export failures to the console.
  /^(Failed to send telemetry to|Error sending telemetry to|Failed to fetch)/,
  // Vue Devtools browser extension warns when multiple versions are installed.
  // Our app is React-only; this noise originates from the user's extensions.
  /^Another version of Vue Devtools/,
];

// True when a console message matches a known third-party pattern we ignore.
export function isIgnoredConsoleMessage(message: unknown): boolean {
  return typeof message === "string" && IGNORED_CONSOLE_MESSAGES.some((pattern) => pattern.test(message));
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
      if (breadcrumb.category === "console" && isIgnoredConsoleMessage(breadcrumb.message)) {
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
