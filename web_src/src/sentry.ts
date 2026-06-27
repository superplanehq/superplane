import * as Sentry from "@sentry/react";
import type { ErrorEvent, EventHint } from "@sentry/react";
import { looksLikeMinifiedReferenceError } from "@/lib/errors";

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
    beforeSend: filterNonActionableErrors,
  });
}

/**
 * Drops Sentry events whose root cause is not actionable from our side, such
 * as `ReferenceError`s about minified bundle identifiers that get raised when
 * a browser extension or content-blocker corrupts our bundle's lexical scope.
 *
 * See `looksLikeMinifiedReferenceError` for the rationale.
 */
export function filterNonActionableErrors(event: ErrorEvent, hint: EventHint): ErrorEvent | null {
  if (isMinifiedReferenceErrorHint(hint) || isMinifiedReferenceErrorEvent(event)) {
    return null;
  }

  return event;
}

function isMinifiedReferenceErrorHint(hint: EventHint): boolean {
  const original = hint.originalException;

  if (original instanceof Error) {
    return looksLikeMinifiedReferenceError(original.message);
  }

  if (typeof original === "string") {
    return looksLikeMinifiedReferenceError(original);
  }

  return false;
}

function isMinifiedReferenceErrorEvent(event: ErrorEvent): boolean {
  const exceptions = event.exception?.values;
  if (!exceptions || exceptions.length === 0) {
    return false;
  }

  return exceptions.some((exception) => {
    if (exception.type && exception.type !== "ReferenceError") {
      return false;
    }

    const value = exception.value;
    return typeof value === "string" && looksLikeMinifiedReferenceError(value);
  });
}

export { Sentry };
