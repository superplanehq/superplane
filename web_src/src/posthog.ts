import posthog from "posthog-js";
import { initializeUtmAttribution } from "@/lib/utmAttribution";

interface PostHogWindow extends Window {
  SUPERPLANE_POSTHOG_KEY?: string;
}

const key = (window as PostHogWindow).SUPERPLANE_POSTHOG_KEY;

if (key) {
  posthog.init(key, {
    api_host: "https://us.i.posthog.com",
    autocapture: false,
    capture_pageview: false,
    person_profiles: "always",
  });
  initializeUtmAttribution(posthog);
}

export { posthog };
export const isPostHogEnabled = !!key;
