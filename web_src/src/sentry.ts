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

// Patterns for known browser-extension noise that should never be reported.
// Cardano wallet extensions (Eternl, Nami, Flint, Yoroi, Lace, Gero, Typhon, …)
// inject themselves into every page and emit console warnings like
// "initEternlDomAPI: href https://…", which `captureConsoleIntegration` would
// otherwise report as warnings to Sentry.
export const ignoredErrorPatterns: (string | RegExp)[] = [/\binit\w+DomAPI\b/];

if (dsn) {
  Sentry.init({
    dsn,
    environment,
    ignoreErrors: ignoredErrorPatterns,
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
