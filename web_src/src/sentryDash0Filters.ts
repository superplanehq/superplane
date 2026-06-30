import type { Breadcrumb, ErrorEvent } from "@sentry/react";

// Dash0 Web SDK logs export failures to the console; captureConsoleIntegration would forward them.
export const DASH0_TELEMETRY_CONSOLE_IGNORE = /^(Failed to send telemetry to|Error sending telemetry to)/;

export function isDash0TelemetryConsoleMessage(message: string): boolean {
  return DASH0_TELEMETRY_CONSOLE_IGNORE.test(message);
}

export function shouldDropDash0TelemetryBreadcrumb(breadcrumb: Breadcrumb): boolean {
  return (
    breadcrumb.category === "console" &&
    typeof breadcrumb.message === "string" &&
    isDash0TelemetryConsoleMessage(breadcrumb.message)
  );
}

function hasDash0TelemetryConsoleBreadcrumb(breadcrumbs: Breadcrumb[] | undefined): boolean {
  return (
    breadcrumbs?.some(
      (breadcrumb) =>
        breadcrumb.category === "console" &&
        typeof breadcrumb.message === "string" &&
        (isDash0TelemetryConsoleMessage(breadcrumb.message) || breadcrumb.message.includes("dash0.com")),
    ) ?? false
  );
}

export function shouldDropDash0TelemetryEvent(event: ErrorEvent): boolean {
  const exception = event.exception?.values?.[0];
  if (!exception) {
    return false;
  }

  const message = exception.value ?? "";
  if (typeof message === "string" && isDash0TelemetryConsoleMessage(message)) {
    return true;
  }

  // captureConsoleIntegration emits the underlying fetch error from console.warn(..., error).
  if (exception.type === "TypeError" && message === "Failed to fetch") {
    return hasDash0TelemetryConsoleBreadcrumb(event.breadcrumbs);
  }

  return false;
}
