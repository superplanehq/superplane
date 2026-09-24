import { afterEach, describe, expect, it, mock } from "bun:test";

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
    const assign = mock();
    const previousAssign = window.location.assign.bind(window.location);
    window.location.assign = assign;

    expect(
      followBrowserAction({
        method: "GET",
        url: "https://github.com/apps/superplane/installations/new?state=abc",
      }),
    ).toBe(true);
    expect(assign).toHaveBeenCalledWith("https://github.com/apps/superplane/installations/new?state=abc");

    window.location.assign = previousAssign;
  });

  it("submits a form for a POST action", () => {
    const submit = mock();
    const previousSubmit = HTMLFormElement.prototype.submit;
    HTMLFormElement.prototype.submit = submit;

    expect(
      followBrowserAction({
        method: "POST",
        url: "https://github.com/settings/apps/new",
        formFields: { manifest: "{}", state: "abc" },
      }),
    ).toBe(true);

    const form = document.querySelector("form");
    expect(form?.getAttribute("method")).toMatch(/post/i);
    expect(form?.action).toContain("https://github.com/settings/apps/new");
    expect(submit).toHaveBeenCalled();

    HTMLFormElement.prototype.submit = previousSubmit;
  });
});
