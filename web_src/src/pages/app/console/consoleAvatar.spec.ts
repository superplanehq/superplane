import { describe, expect, it } from "vitest";

import { resolveConsoleAvatar } from "./consoleAvatar";

describe("resolveConsoleAvatar", () => {
  it("uses the GitHub avatar for a plain username string", () => {
    expect(resolveConsoleAvatar("forestileao")).toEqual({
      src: "https://github.com/forestileao.png",
      name: "forestileao",
      initials: "F",
    });
  });

  it("passes direct image URLs through as the avatar source", () => {
    expect(resolveConsoleAvatar("https://github.com/forestileao.png?size=64")).toEqual({
      src: "https://github.com/forestileao.png?size=64",
      name: "",
    });
  });

  it("uses the GitHub avatar when author.username is present", () => {
    expect(resolveConsoleAvatar({ name: "Pedro Leão", username: "forestileao" }, { name: "Pedro Leão" })).toEqual({
      src: "https://github.com/forestileao.png",
      name: "Pedro Leão",
      initials: "P",
    });
  });

  it("provides an initials fallback alongside the image so a 404 avatar degrades gracefully", () => {
    // Bot accounts (e.g. `...-integration-9000[bot]`) have no avatar at
    // `github.com/<name>.png`, so the image 404s and the UI must fall back.
    const resolved = resolveConsoleAvatar({ username: "superplane-gh-integration-9000[bot]" });
    expect(resolved.src).toBe("https://github.com/superplane-gh-integration-9000[bot].png");
    expect(resolved.initials).toBe("S");
  });

  it("falls back to initials when no username is available", () => {
    expect(resolveConsoleAvatar({ name: "cloud-robot", email: "bot@example.com" }, { name: "cloud-robot" })).toEqual({
      initials: "C",
      name: "cloud-robot",
    });
  });
});
