import { describe, expect, it } from "vitest";
import { getAuthRedirectURL, getWelcomeRedirectPath } from "./authRedirect";

describe("auth redirect helpers", () => {
  it("reads explicit JSON redirect URLs", async () => {
    const response = new Response(JSON.stringify({ redirectUrl: "/welcome" }), {
      headers: { "Content-Type": "application/json" },
    });

    expect(await getAuthRedirectURL(response)).toBe("/welcome");
  });

  it("falls back to the response URL when no JSON redirect is present", async () => {
    const response = new Response("", {
      headers: { "Content-Type": "text/html" },
    });

    Object.defineProperty(response, "url", {
      value: "http://localhost/canvases",
    });

    expect(await getAuthRedirectURL(response)).toBe("http://localhost/canvases");
  });

  it("builds same-origin welcome redirect paths", () => {
    expect(getWelcomeRedirectPath("/welcome", "/invite/token-123")).toBe("/welcome?redirect=%2Finvite%2Ftoken-123");
  });

  it("rejects non-welcome redirect paths", () => {
    expect(getWelcomeRedirectPath("/org-123", "")).toBeNull();
  });
});
