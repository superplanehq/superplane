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

// Patterns matched against the event message and exception value to drop noise
// emitted by third-party browser extensions (wallets, content scripts, etc.).
// These show up because `captureConsoleIntegration` forwards `console.warn`/
// `console.error` calls — including those originating from extensions — to
// Sentry, even though they are unrelated to the SuperPlane app.
export const ignoreErrors: (string | RegExp)[] = [
  // Eternl Cardano wallet extension noise (e.g. "initEternlDomAPI: domId ... false")
  /initEternlDomAPI/i,
  // Other common Cardano / crypto wallet extensions
  /initCardanoDomAPI/i,
  /initNamiDomAPI/i,
  /initFlintDomAPI/i,
  /initTyphonDomAPI/i,
  /initYoroiDomAPI/i,
  /initLaceDomAPI/i,
  /initNuFiDomAPI/i,
  /initGeroDomAPI/i,
  /initBegonDomAPI/i,
  // Generic browser extension internals
  /Extension context invalidated/i,
  /chrome-extension:\/\//i,
  /moz-extension:\/\//i,
  /safari-(web-)?extension:\/\//i,
  // ResizeObserver loop notifications are benign and emitted by Chrome.
  /ResizeObserver loop limit exceeded/i,
  /ResizeObserver loop completed with undelivered notifications/i,
];

// Stack-trace URL prefixes that indicate the event originated from a browser
// extension rather than the SuperPlane app.
export const denyUrls: (string | RegExp)[] = [
  /^chrome-extension:\/\//i,
  /^moz-extension:\/\//i,
  /^safari-(web-)?extension:\/\//i,
];

if (dsn) {
  Sentry.init({
    dsn,
    environment,
    ignoreErrors,
    denyUrls,
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
