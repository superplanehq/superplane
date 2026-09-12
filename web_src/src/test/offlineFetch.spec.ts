import { describe, expect, it } from "bun:test";

describe("Happy DOM network interceptor", () => {
  it("returns 404 for an unmocked http fetch instead of opening a socket", async () => {
    const response = await fetch("http://127.0.0.1/api/v1/me");

    expect(response.status).toBe(404);
    expect(response.ok).toBe(false);
  });

  it("still serves data URLs from Happy DOM", async () => {
    const response = await fetch("data:text/plain,hello");

    expect(response.ok).toBe(true);
    expect(await response.text()).toBe("hello");
  });
});
