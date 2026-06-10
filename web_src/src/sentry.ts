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

// Patterns for messages that originate from third-party browser extensions
// (content scripts, ad blockers, password managers, etc.) and should never be
// reported to our Sentry project. The `captureConsoleIntegration` forwards
// `console.warn` / `console.error` calls — including ones made by extensions
// running in the page — so without this list extensions like 1Blocker pollute
// our Sentry inbox with noise such as
// "[1Blocker] Duplicate content script injection blocked".
export const BROWSER_EXTENSION_IGNORE_PATTERNS: RegExp[] = [
  /\[1Blocker\]/i,
  /\[AdGuard\]/i,
  /\[uBlock\]/i,
  /\bAdGuardAssistant\b/i,
  /\bAdblock(?:Plus)?\b/i,
  /\bGhostery\b/i,
  /\bLastPass\b/i,
  /\b1Password\b/i,
  /\bBitwarden\b/i,
  /\bGrammarly\b/i,
  /\bMetaMask\b/i,
  /Duplicate content script injection blocked/i,
];

export const BROWSER_EXTENSION_DENY_URLS: RegExp[] = [
  /^chrome:\/\//i,
  /^chrome-extension:\/\//i,
  /^moz-extension:\/\//i,
  /^safari-extension:\/\//i,
  /^safari-web-extension:\/\//i,
  /^webkit-masked-url:\/\//i,
  /^edge:\/\//i,
  /^extensions\//i,
];

if (dsn) {
  Sentry.init({
    dsn,
    environment,
    ignoreErrors: BROWSER_EXTENSION_IGNORE_PATTERNS,
    denyUrls: BROWSER_EXTENSION_DENY_URLS,
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
