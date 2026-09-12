import { init } from "@dash0/sdk-web";
import { pageObservabilityMetadata } from "@/lib/dash0Observability";

interface Dash0Window extends Window {
  SUPERPLANE_DASH0_OTLP_ENDPOINT?: string;
  SUPERPLANE_DASH0_AUTH_TOKEN?: string;
  SUPERPLANE_DASH0_SERVICE_NAME?: string;
  SUPERPLANE_DASH0_ENVIRONMENT?: string;
}

const dash0IgnoredUrls = [/\/ws\//, /posthog\.com/];

function dash0Window(): Dash0Window | undefined {
  return typeof window !== "undefined" ? (window as Dash0Window) : undefined;
}

export function dash0InitOptions(win: Dash0Window | undefined = dash0Window()) {
  const endpointUrl = win?.SUPERPLANE_DASH0_OTLP_ENDPOINT?.trim();
  const authToken = win?.SUPERPLANE_DASH0_AUTH_TOKEN?.trim();
  if (!endpointUrl || !authToken) {
    return null;
  }

  return {
    serviceName: win?.SUPERPLANE_DASH0_SERVICE_NAME?.trim() || "superplane-web",
    environment: win?.SUPERPLANE_DASH0_ENVIRONMENT?.trim() || undefined,
    endpoint: {
      url: endpointUrl,
      authToken,
    },
    ignoreUrls: dash0IgnoredUrls,
    pageViewInstrumentation: {
      generateMetadata: (url: URL) => pageObservabilityMetadata(url.pathname),
    },
  };
}

/** Read window flags and start Dash0. Safe to call again after tests change the flags. */
export function initDash0(win: Dash0Window | undefined = dash0Window()): boolean {
  const options = dash0InitOptions(win);
  if (!options) {
    return false;
  }
  init(options);
  return true;
}

export const isDash0Enabled = initDash0();
