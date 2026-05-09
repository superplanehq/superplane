import * as Sentry from "@sentry/react";

interface SentryWindow extends Window {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
}

// Patterns for noisy errors injected by browser extensions or third-party
// scripts (e.g. the Telegram Mini App SDK's `postEvent` bridge) that surface
// inside Sentry but do not originate from SuperPlane code. Keeping them out
// of Sentry avoids alert noise without hiding real application errors.
//
// See: https://github.com/getsentry/sentry-javascript/issues/16329
export const IGNORED_ERROR_PATTERNS: (string | RegExp)[] = [
  /Error invoking postEvent: Method not found/i,
  /^postEvent: .*Method not found/i,
];

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
    ignoreErrors: IGNORED_ERROR_PATTERNS,
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
