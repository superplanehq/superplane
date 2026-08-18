const BASE_DELAY_MS = 1000;
const MAX_DELAY_MS = 30000;

/**
 * Exponential backoff with jitter for websocket reconnect attempts.
 *
 * Meant to be passed directly as `reconnectInterval` to react-use-websocket,
 * which calls it with the 1-indexed attempt number and waits the returned
 * number of milliseconds before retrying. The attempt counter resets on
 * `onopen`, so a brief blip still reconnects in about a second while a
 * sustained outage backs off to a 30s cap instead of retrying every 3s
 * forever.
 *
 * Jitter is "equal jitter" (half fixed, half random) rather than full jitter
 * so clients that were disconnected by the same event don't stay phase-locked
 * and slam the backend in the same instant once it recovers, while still
 * guaranteeing a minimum spacing between attempts at each backoff step.
 */
export function getWebsocketReconnectInterval(attemptNumber: number): number {
  const exponentialDelay = Math.min(BASE_DELAY_MS * 2 ** (attemptNumber - 1), MAX_DELAY_MS);
  return exponentialDelay / 2 + Math.random() * (exponentialDelay / 2);
}
