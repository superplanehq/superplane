import posthog from "posthog-js";

interface PostHogWindow extends Window {
  SUPERPLANE_POSTHOG_KEY?: string;
}

const key = (window as PostHogWindow).SUPERPLANE_POSTHOG_KEY;

if (key) {
  posthog.init(key, {
    api_host: "https://us.i.posthog.com",
    autocapture: false,
    capture_pageview: false,
  });
}

export { posthog };
