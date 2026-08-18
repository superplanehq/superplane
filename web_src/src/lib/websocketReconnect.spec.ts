import { afterEach, describe, expect, it } from "vitest";
import { getWebsocketReconnectInterval } from "@/lib/websocketReconnect";

describe("getWebsocketReconnectInterval", () => {
  const originalRandom = Math.random;

  afterEach(() => {
    Math.random = originalRandom;
  });

  it("reconnects the first attempt in about a second", () => {
    Math.random = () => 0;
    expect(getWebsocketReconnectInterval(1)).toBe(500);

    Math.random = () => 1;
    expect(getWebsocketReconnectInterval(1)).toBe(1000);
  });

  it("doubles the delay window with each attempt", () => {
    Math.random = () => 0;
    expect(getWebsocketReconnectInterval(1)).toBe(500);
    expect(getWebsocketReconnectInterval(2)).toBe(1000);
    expect(getWebsocketReconnectInterval(3)).toBe(2000);
    expect(getWebsocketReconnectInterval(4)).toBe(4000);
  });

  it("caps the delay at 30s and never grows further on sustained outages", () => {
    Math.random = () => 1;
    expect(getWebsocketReconnectInterval(6)).toBe(30000);
    expect(getWebsocketReconnectInterval(20)).toBe(30000);
    expect(getWebsocketReconnectInterval(1000)).toBe(30000);
  });

  it("spreads simultaneous clients instead of retrying in lockstep", () => {
    const seen = new Set<number>();
    for (let i = 0; i < 50; i++) {
      seen.add(getWebsocketReconnectInterval(4));
    }
    // With real randomness, 50 draws from a continuous range should not all
    // collapse onto the same value the way a fixed interval would.
    expect(seen.size).toBeGreaterThan(1);
  });

  it("never returns a delay below the guaranteed minimum for a given attempt", () => {
    Math.random = () => 0;
    const attempt = 5; // exponential delay = 16000ms
    expect(getWebsocketReconnectInterval(attempt)).toBe(8000);
  });
});
