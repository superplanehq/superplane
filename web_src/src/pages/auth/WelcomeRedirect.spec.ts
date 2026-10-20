import { describe, expect, it } from "bun:test";
import { getWelcomeRedirectPath } from "./welcomeRedirectPath";

describe("getWelcomeRedirectPath", () => {
  it("returns a safe internal redirect", () => {
    expect(getWelcomeRedirectPath("/invite/token-123")).toBe("/invite/token-123");
  });

  it("returns home when redirect is missing", () => {
    expect(getWelcomeRedirectPath(null)).toBe("/");
  });

  it("rejects external redirects", () => {
    expect(getWelcomeRedirectPath("https://example.com")).toBe("/");
  });

  it("rejects protocol-relative redirects", () => {
    expect(getWelcomeRedirectPath("//example.com")).toBe("/");
  });

  it("rejects recursive welcome redirects", () => {
    expect(getWelcomeRedirectPath("/welcome?redirect=/invite/token-123")).toBe("/");
  });
});
