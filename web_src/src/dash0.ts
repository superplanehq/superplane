import { init } from "@dash0/sdk-web";
import { pageObservabilityMetadata } from "@/lib/dash0Observability";

interface Dash0Window extends Window {
  SUPERPLANE_DASH0_OTLP_ENDPOINT?: string;
  SUPERPLANE_DASH0_AUTH_TOKEN?: string;
  SUPERPLANE_DASH0_SERVICE_NAME?: string;
  SUPERPLANE_DASH0_ENVIRONMENT?: string;
}

const dash0Window = typeof window !== "undefined" ? (window as Dash0Window) : undefined;
const endpointUrl = dash0Window?.SUPERPLANE_DASH0_OTLP_ENDPOINT?.trim();
const authToken = dash0Window?.SUPERPLANE_DASH0_AUTH_TOKEN?.trim();

export const isDash0Enabled = !!(endpointUrl && authToken);

const dash0IgnoredUrls = [/\/ws\//, /posthog\.com/];

if (endpointUrl && authToken) {
  init({
    serviceName: dash0Window?.SUPERPLANE_DASH0_SERVICE_NAME?.trim() || "superplane-web",
    environment: dash0Window?.SUPERPLANE_DASH0_ENVIRONMENT?.trim() || undefined,
    endpoint: {
      url: endpointUrl,
      authToken,
    },
    ignoreUrls: dash0IgnoredUrls,
    pageViewInstrumentation: {
      generateMetadata: (url) => pageObservabilityMetadata(url.pathname),
    },
  });
}
