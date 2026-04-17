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
    session_recording: {
      // Record the React Flow canvas at low FPS
      captureCanvas: {
        recordCanvas: true,
        canvasFps: 1,
        canvasQuality: "0.1",
      },
    },
  });
}

export { posthog };
