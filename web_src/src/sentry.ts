import * as Sentry from "@sentry/react";
import type { ErrorEvent, EventHint } from "@sentry/react";
import { looksLikeBrowserNetworkError } from "@/lib/errors";

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
    beforeSend: filterBrowserNetworkErrors,
  });
}

/**
 * Drops `TypeError: Failed to fetch` (and equivalents on other browsers, such
 * as Safari's `TypeError: Load failed`) from Sentry reports. These come from
 * user-side connectivity issues — offline, intermittent network, captive
 * portals, ad-blockers — and the app already handles them gracefully in the
 * UI, so reporting them just creates noise.
 */
export function filterBrowserNetworkErrors(event: ErrorEvent, hint: EventHint): ErrorEvent | null {
  if (isBrowserNetworkErrorHint(hint) || isBrowserNetworkErrorEvent(event)) {
    return null;
  }

  return event;
}

function isBrowserNetworkErrorHint(hint: EventHint): boolean {
  const original = hint.originalException;
  if (original instanceof Error) {
    return looksLikeBrowserNetworkError(original.message);
  }

  if (typeof original === "string") {
    return looksLikeBrowserNetworkError(original);
  }

  return false;
}

function isBrowserNetworkErrorEvent(event: ErrorEvent): boolean {
  const exceptions = event.exception?.values;
  if (!exceptions || exceptions.length === 0) {
    return false;
  }

  return exceptions.some((exception) => {
    const value = exception.value;
    return typeof value === "string" && looksLikeBrowserNetworkError(value);
  });
}

export { Sentry };
