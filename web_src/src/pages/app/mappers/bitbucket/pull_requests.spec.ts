import { describe, expect, it } from "bun:test";
import { pullRequestDetailFields } from "./pull_requests";

describe("pullRequestDetailFields", () => {
  it("shows the Bitbucket pull request ID, branches and URL", () => {
    expect(
      pullRequestDetailFields({
        id: 42,
        title: "feat: Retry",
        state: "OPEN",
        source: { branch: { name: "feat/retry" } },
        destination: { branch: { name: "main" } },
        links: { html: { href: "https://bitbucket.org/acme/widgets/pull-requests/42" } },
      }),
    ).toEqual({
      "Pull Request": "#42",
      Title: "feat: Retry",
      State: "OPEN",
      Branches: "feat/retry → main",
      "Pull Request URL": "https://bitbucket.org/acme/widgets/pull-requests/42",
    });
  });

  it("marks a draft and leaves out empty fields", () => {
    expect(pullRequestDetailFields({ id: 7, state: "OPEN", draft: true })).toEqual({
      "Pull Request": "#7",
      State: "OPEN (draft)",
    });
  });
});
