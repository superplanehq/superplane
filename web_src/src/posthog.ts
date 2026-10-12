import posthog from "posthog-js";
import { initializeUtmAttribution } from "@/lib/utmAttribution";

interface PostHogWindow extends Window {
  SUPERPLANE_POSTHOG_KEY?: string;
}

function posthogWindow(): PostHogWindow {
  return window as PostHogWindow;
}

/** Read the window key and start PostHog. Safe to call again after tests change the key. */
export function initPostHog(win: PostHogWindow = posthogWindow()): boolean {
  const key = win.SUPERPLANE_POSTHOG_KEY;
  if (!key) {
    return false;
  }

  posthog.init(key, {
    api_host: "https://us.i.posthog.com",
    autocapture: false,
    capture_pageview: false,
    person_profiles: "always",
  });
  initializeUtmAttribution(posthog);
  return true;
}

initPostHog();

export { posthog };

export function isPostHogEnabled(win: PostHogWindow = posthogWindow()): boolean {
  return !!win.SUPERPLANE_POSTHOG_KEY;
}
