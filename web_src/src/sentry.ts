import * as Sentry from "@sentry/react";

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

// Patterns for errors that originate from third-party browser extensions or
// other non-actionable noise we do not want to capture in Sentry.
// See: https://docs.sentry.io/platforms/javascript/configuration/filtering/#using-ignoreerrors
const ignoreErrors: (string | RegExp)[] = [
  // Microsoft Outlook / Office SafeLink browser extension. The extension's
  // content script posts messages to a host object that may no longer exist
  // and surfaces as an unhandled promise rejection in our app.
  // https://github.com/getsentry/sentry-javascript/issues/3440
  /Object Not Found Matching Id:\d+, MethodName:\w+, ParamCount:\d+/,
  // Generic ResizeObserver loop warning emitted by some browsers / extensions.
  "ResizeObserver loop limit exceeded",
  "ResizeObserver loop completed with undelivered notifications",
];

if (dsn) {
  Sentry.init({
    dsn,
    environment,
    ignoreErrors,
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
