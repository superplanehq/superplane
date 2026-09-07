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

// True when the event is the known-benign monaco-editor "Canceled"
// unhandled rejection. monaco-editor (CDN-loaded) rejects a DeferredPromise
// with no .catch() when a WebKit clipboard-write is superseded by a new copy
// action. This is upstream noise, not a bug in our code.
export function isMonacoCanceledEvent(event: Sentry.ErrorEvent): boolean {
  const exception = event.exception?.values?.[0];
  if (exception?.value !== "Canceled") {
    return false;
  }

  const frames = exception.stacktrace?.frames ?? [];
  return (
    frames.length > 0 &&
    frames.every((frame) => frame.filename?.includes("monaco-editor") || frame.filename?.includes("@sentry"))
  );
}

// Matches a UUID (with or without dashes normalized) anywhere in a path
// segment, so a UUID organization ID normalizes the same way as a slug.
const UUID_SEGMENT = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi;

// Matches an HTTP status code embedded in a log message, e.g.
// "GET /api/v1/organizations/superplane/byok-models failed: 500".
const HTTP_STATUS_IN_MESSAGE = /\b([45]\d{2})\b/;

// Extracts the first "/api/v1/..." path from free-form text (a log message
// or a full URL), stripping any query string.
function extractApiPath(text: string): string | undefined {
  const match = text.match(/\/api\/v1\/[^\s"')]+/);
  if (!match) {
    return undefined;
  }

  return match[0].split("?")[0];
}

// Builds a stable route template for an "/api/v1/..." path by replacing
// dynamic identifier segments (organization slug or UUID, and any other
// UUID segment) with placeholders. Returns undefined for paths that are not
// under /api/v1/, so unrelated events are left untouched.
export function normalizeApiRouteTemplate(path: string): string | undefined {
  if (!path.startsWith("/api/v1/")) {
    return undefined;
  }

  const organizationsMatch = path.match(/^\/api\/v1\/organizations\/([^/]+)(\/.*)?$/);
  if (organizationsMatch) {
    return `/api/v1/organizations/{org}${organizationsMatch[2] ?? ""}`;
  }

  return path.replace(UUID_SEGMENT, "{id}");
}

// Finds the "/api/v1/..." path an event is about, preferring the structured
// request URL and falling back to scanning the message text. This covers
// both a captured HTTP exception (event.request.url) and a console.error
// message event produced by captureConsoleIntegration, which only carries
// the URL as text (see the Sentry issue this fixes: PRODUCTION-93/97).
function findEventApiPath(event: Sentry.Event): string | undefined {
  const requestUrl = event.request?.url;
  if (requestUrl) {
    const path = extractApiPath(requestUrl);
    if (path) {
      return path;
    }
  }

  if (event.message) {
    return extractApiPath(event.message);
  }

  return undefined;
}

// Sets a stable fingerprint on API HTTP-error events so a slug URL and a
// UUID URL for the same route (e.g. GET /api/v1/organizations/superplane/
// byok-models vs GET /api/v1/organizations/<uuid>/byok-models) group into one
// Sentry issue instead of two. Events that already have an explicit
// fingerprint, or that are not about an "/api/v1/..." route, are returned
// unchanged so unrelated grouping is preserved.
export function normalizeSentryFingerprint<E extends Sentry.Event>(event: E): E {
  if (event.fingerprint) {
    return event;
  }

  const path = findEventApiPath(event);
  if (!path) {
    return event;
  }

  const routeTemplate = normalizeApiRouteTemplate(path);
  if (!routeTemplate) {
    return event;
  }

  const method = event.request?.method ?? "UNKNOWN";
  const statusMatch = event.message?.match(HTTP_STATUS_IN_MESSAGE);
  const status = statusMatch ? statusMatch[1] : "error";

  return {
    ...event,
    fingerprint: ["http-error", method, routeTemplate, status],
  } as E;
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

      if (isMonacoCanceledEvent(event)) {
        return null;
      }

      return normalizeSentryFingerprint(event);
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
