import { describe, expect, it } from "vitest";

import { resolveConsoleAvatar } from "./consoleAvatar";

describe("resolveConsoleAvatar", () => {
  it("uses the GitHub avatar for a plain username string", () => {
    expect(resolveConsoleAvatar("forestileao")).toEqual({
      src: "https://github.com/forestileao.png",
      name: "forestileao",
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
    });
  });

  it("falls back to initials when no username is available", () => {
    expect(resolveConsoleAvatar({ name: "cloud-robot", email: "bot@example.com" }, { name: "cloud-robot" })).toEqual({
      initials: "C",
      name: "cloud-robot",
    });
  });
});
