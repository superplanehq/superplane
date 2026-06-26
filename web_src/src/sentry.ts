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

// Console messages emitted by the Dash0 web SDK when it cannot deliver its own
// telemetry. We capture browser warnings in Sentry, but these particular ones
// describe failures of our observability stack itself (not application bugs),
// so reporting them creates a noisy feedback loop where a single bad telemetry
// request can fan out into many Sentry warnings.
const dash0TelemetryNoisePatterns: RegExp[] = [
  /Failed to send telemetry to /,
  /Error sending telemetry to /,
  /Unable to send telemetry, fetch is not defined/,
  /Failed to transmit (logs|spans)/,
];

if (dsn) {
  Sentry.init({
    dsn,
    environment,
    ignoreErrors: dash0TelemetryNoisePatterns,
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
