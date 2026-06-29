import type { EventHint, ErrorEvent } from "@sentry/react";

// Console messages emitted by the Dash0 web SDK when it cannot deliver its own
// telemetry. We capture browser warnings in Sentry, but these describe failures
// of our observability stack itself (not application bugs), so reporting them
// creates a noisy feedback loop where a single bad telemetry request fans out
// into many Sentry alerts.
export const dash0TelemetryNoisePatterns: RegExp[] = [
  /Failed to send telemetry to /,
  /Error sending telemetry to /,
  /Unable to send telemetry, fetch is not defined/,
  /Failed to transmit (logs|spans)/,
];

function getConsoleArguments(event: ErrorEvent, hint?: EventHint): unknown[] | undefined {
  const extraArgs = event.extra?.arguments;
  if (Array.isArray(extraArgs)) {
    return extraArgs;
  }

  const hintArgs = (hint?.captureContext as { extra?: { arguments?: unknown[] } } | undefined)?.extra?.arguments;
  if (Array.isArray(hintArgs)) {
    return hintArgs;
  }

  return undefined;
}

function matchesDash0TelemetryNoise(text: string): boolean {
  return dash0TelemetryNoisePatterns.some((pattern) => pattern.test(text));
}

function hasDash0TelemetryConsoleContext(event: ErrorEvent, hint?: EventHint): boolean {
  const mechanismType = event.exception?.values?.[0]?.mechanism?.type;
  if (mechanismType !== "console") {
    return false;
  }

  const consoleArgs = getConsoleArguments(event, hint);
  if (!consoleArgs?.length) {
    return false;
  }

  const joinedArgs = consoleArgs.map(String).join(" ");
  if (matchesDash0TelemetryNoise(joinedArgs)) {
    return true;
  }

  return consoleArgs.some((arg) => typeof arg === "string" && matchesDash0TelemetryNoise(arg));
}

export function isDash0TelemetryNoiseEvent(event: ErrorEvent, hint?: EventHint): boolean {
  const message = event.message ?? event.exception?.values?.[0]?.value ?? "";
  if (typeof message === "string" && matchesDash0TelemetryNoise(message)) {
    return true;
  }

  return hasDash0TelemetryConsoleContext(event, hint);
}
