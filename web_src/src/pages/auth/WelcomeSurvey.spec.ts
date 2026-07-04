import { describe, expect, it } from "vitest";
import { getWelcomeSurveyRedirectPath } from "./welcomeSurveyRedirect";

describe("getWelcomeSurveyRedirectPath", () => {
  it("returns a safe internal redirect", () => {
    expect(getWelcomeSurveyRedirectPath("/invite/token-123")).toBe("/invite/token-123");
  });

  it("returns home when redirect is missing", () => {
    expect(getWelcomeSurveyRedirectPath(null)).toBe("/");
  });

  it("rejects external redirects", () => {
    expect(getWelcomeSurveyRedirectPath("https://example.com")).toBe("/");
  });

  it("rejects protocol-relative redirects", () => {
    expect(getWelcomeSurveyRedirectPath("//example.com")).toBe("/");
  });

  it("rejects welcome redirects", () => {
    expect(getWelcomeSurveyRedirectPath("/welcome?redirect=/invite/token-123")).toBe("/");
  });
});
