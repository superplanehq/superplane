import { describe, expect, it } from "bun:test";

import { githubAvatarImageUrl, githubPersonAvatarUrl } from "./githubAvatar";

describe("githubPersonAvatarUrl", () => {
  it("uses github.com for a username when the host is empty", () => {
    expect(githubPersonAvatarUrl("forestileao")).toBe("https://github.com/forestileao.png");
    expect(githubAvatarImageUrl("forestileao")).toBe("https://github.com/forestileao.png");
  });

  it("prefers the avatar URL GitHub already sent", () => {
    expect(
      githubPersonAvatarUrl({
        username: "alex",
        avatar_url: "https://avatars.acme.ghe.com/u/12?v=4",
      }),
    ).toBe("https://avatars.acme.ghe.com/u/12?v=4");
  });

  it("builds the tenant avatar from html_url when GitHub omits avatar_url", () => {
    expect(
      githubPersonAvatarUrl({
        login: "alex",
        html_url: "https://acme.ghe.com/alex",
      }),
    ).toBe("https://acme.ghe.com/alex.png");
  });

  it("ignores a non-http avatar_url", () => {
    expect(githubPersonAvatarUrl({ username: "alex", avatar_url: "javascript:alert(1)" })).toBe(
      "https://github.com/alex.png",
    );
  });

  it("uses the tenant host for a GitHub push author", () => {
    const author = {
      date: "2026-10-21T12:00:00Z",
      email: "alex@acme.com",
      name: "Alex Rivera",
      username: "alex",
    };
    expect(githubPersonAvatarUrl(author)).toBe("https://github.com/alex.png");
    expect(githubPersonAvatarUrl(author, "acme.ghe.com")).toBe("https://acme.ghe.com/alex.png");
  });
});
