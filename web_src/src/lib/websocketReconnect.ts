/**
 * Reconnect pacing for the app's websockets.
 *
 * Every open canvas and agent session reconnects when the backend restarts, so
 * a fixed interval makes all of them retry in lockstep for as long as the
 * backend is unavailable. Delays grow exponentially up to a cap and carry
 * jitter, which spreads the retries out and keeps a slow restart from being
 * hammered by every open tab.
 */

export const WEBSOCKET_RECONNECT_BASE_MS = 1_000;
export const WEBSOCKET_RECONNECT_MAX_MS = 30_000;

/**
 * Delay before the next reconnect attempt, in milliseconds.
 *
 * `attempt` is zero-based: 0 is the first retry after a connection drops. The
 * counter resets once a connection opens, so a short blip still reconnects
 * quickly.
 */
export function getWebsocketReconnectDelay(attempt: number, random: () => number = Math.random): number {
  const safeAttempt = Number.isFinite(attempt) ? Math.max(0, Math.floor(attempt)) : 0;
  const exponential = Math.min(WEBSOCKET_RECONNECT_BASE_MS * 2 ** safeAttempt, WEBSOCKET_RECONNECT_MAX_MS);

  // Half of the delay is fixed and half is jittered, so retries stay bounded
  // while still being spread across clients.
  return Math.round(exponential / 2 + (exponential / 2) * random());
}
