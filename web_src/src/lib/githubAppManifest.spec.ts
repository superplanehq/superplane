import { afterEach, describe, expect, it, mock } from "bun:test";

import { startPublicGitHubAppCreate } from "./githubAppManifest";

describe("startPublicGitHubAppCreate", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("posts the GitHub App manifest form", async () => {
    const submit = mock();
    const previousSubmit = HTMLFormElement.prototype.submit;
    HTMLFormElement.prototype.submit = submit;
    const previousFetch = globalThis.fetch;
    globalThis.fetch = mock(async () => {
      return new Response(
        JSON.stringify({
          url: "https://github.com/settings/apps/new",
          method: "POST",
          form: { manifest: "{}", state: "abc" },
        }),
        { status: 200 },
      );
    }) as typeof fetch;

    await expect(startPublicGitHubAppCreate("/org/workspaces/new/setup")).resolves.toBe(true);
    expect(document.querySelector("form")?.action).toContain("https://github.com/settings/apps/new");
    expect(submit).toHaveBeenCalled();

    HTMLFormElement.prototype.submit = previousSubmit;
    globalThis.fetch = previousFetch;
  });
});
