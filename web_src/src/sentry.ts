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

/**
 * Returns true when the given value looks like a cancellation/abort error that
 * we intentionally do not want to surface to Sentry.
 *
 * Most notably, Monaco Editor throws a `CancellationError` whose `name` and
 * `message` are both the literal string `"Canceled"` whenever an in-flight
 * editor request is cancelled (e.g. because the user navigated away or the
 * editor was disposed). Monaco filters these internally, but they still
 * propagate to the global `unhandledrejection` handler that Sentry hooks into,
 * producing noisy "Canceled: Canceled" issues. We also drop standard DOM
 * `AbortError`s for the same reason.
 */
export function isIgnoredCancellationError(value: unknown): boolean {
  if (!value || typeof value !== "object") {
    return false;
  }
  const name = (value as { name?: unknown }).name;
  const message = (value as { message?: unknown }).message;
  if (typeof name !== "string") {
    return false;
  }
  if (name === "Canceled" && message === "Canceled") {
    return true;
  }
  if (name === "AbortError") {
    return true;
  }
  return false;
}

function eventLooksLikeCancellation(event: Sentry.ErrorEvent): boolean {
  const values = event.exception?.values;
  if (!values || values.length === 0) {
    return false;
  }
  return values.some((entry) => {
    if (entry.type === "Canceled" && entry.value === "Canceled") {
      return true;
    }
    if (entry.type === "AbortError") {
      return true;
    }
    return false;
  });
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
    ignoreErrors: ["Canceled: Canceled"],
    beforeSend(event, hint) {
      if (isIgnoredCancellationError(hint?.originalException)) {
        return null;
      }
      if (eventLooksLikeCancellation(event)) {
        return null;
      }
      return event;
    },
  });
}

export { Sentry };
