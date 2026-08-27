import { afterEach, describe, expect, it, vi } from "vitest";

import { followBrowserAction } from "./browserAction";

describe("followBrowserAction", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("returns false when the action has no URL", () => {
    expect(followBrowserAction(undefined)).toBe(false);
    expect(followBrowserAction({})).toBe(false);
  });

  it("assigns the location for a GET action", () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { assign });

    expect(
      followBrowserAction({
        method: "GET",
        url: "https://github.com/apps/superplane/installations/new?state=abc",
      }),
    ).toBe(true);
    expect(assign).toHaveBeenCalledWith("https://github.com/apps/superplane/installations/new?state=abc");

    vi.unstubAllGlobals();
  });

  it("submits a form for a POST action", () => {
    const submit = vi.fn();
    HTMLFormElement.prototype.submit = submit;

    expect(
      followBrowserAction({
        method: "POST",
        url: "https://github.com/settings/apps/new",
        formFields: { manifest: "{}", state: "abc" },
      }),
    ).toBe(true);

    const form = document.querySelector("form");
    expect(form?.method).toMatch(/post/i);
    expect(form?.action).toContain("https://github.com/settings/apps/new");
    expect(submit).toHaveBeenCalled();
  });
});
