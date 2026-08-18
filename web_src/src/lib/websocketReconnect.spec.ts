import { describe, expect, it } from "vitest";
import {
  WEBSOCKET_RECONNECT_BASE_MS,
  WEBSOCKET_RECONNECT_MAX_MS,
  getWebsocketReconnectDelay,
} from "@/lib/websocketReconnect";

describe("getWebsocketReconnectDelay", () => {
  it("backs off exponentially", () => {
    const noJitter = () => 0;

    expect(getWebsocketReconnectDelay(0, noJitter)).toBe(WEBSOCKET_RECONNECT_BASE_MS / 2);
    expect(getWebsocketReconnectDelay(1, noJitter)).toBe(WEBSOCKET_RECONNECT_BASE_MS);
    expect(getWebsocketReconnectDelay(2, noJitter)).toBe(WEBSOCKET_RECONNECT_BASE_MS * 2);
    expect(getWebsocketReconnectDelay(3, noJitter)).toBe(WEBSOCKET_RECONNECT_BASE_MS * 4);
  });

  it("retries quickly on the first attempts", () => {
    // A short blip should not leave the canvas stale for long.
    expect(getWebsocketReconnectDelay(0, () => 1)).toBeLessThanOrEqual(WEBSOCKET_RECONNECT_BASE_MS);
    expect(getWebsocketReconnectDelay(1, () => 1)).toBeLessThanOrEqual(2 * WEBSOCKET_RECONNECT_BASE_MS);
  });

  it("caps the delay", () => {
    for (const attempt of [10, 50, 1_000, Number.MAX_SAFE_INTEGER]) {
      expect(getWebsocketReconnectDelay(attempt, () => 1)).toBe(WEBSOCKET_RECONNECT_MAX_MS);
      expect(getWebsocketReconnectDelay(attempt, () => 0)).toBe(WEBSOCKET_RECONNECT_MAX_MS / 2);
    }
  });

  it("jitters so that clients do not retry in lockstep", () => {
    const delays = new Set<number>();
    for (let i = 0; i < 200; i++) {
      delays.add(getWebsocketReconnectDelay(3));
    }

    expect(delays.size).toBeGreaterThan(50);
    for (const delay of delays) {
      expect(delay).toBeGreaterThanOrEqual(WEBSOCKET_RECONNECT_BASE_MS * 4);
      expect(delay).toBeLessThanOrEqual(WEBSOCKET_RECONNECT_BASE_MS * 8);
    }
  });

  it("never returns a negative or non-finite delay", () => {
    for (const attempt of [-5, Number.NaN, Number.POSITIVE_INFINITY, 0.5]) {
      const delay = getWebsocketReconnectDelay(attempt);

      expect(Number.isFinite(delay)).toBe(true);
      expect(delay).toBeGreaterThan(0);
      expect(delay).toBeLessThanOrEqual(WEBSOCKET_RECONNECT_MAX_MS);
    }
  });
});
