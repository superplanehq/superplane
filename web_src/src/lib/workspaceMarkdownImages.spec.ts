import { afterEach, describe, expect, it } from "bun:test";

import {
  clearWorkspaceMarkdownImageLoadCache,
  forgetWorkspaceMarkdownImageLoad,
  rememberWorkspaceMarkdownImageLoad,
  workspaceMarkdownImageIsLoaded,
} from "./workspaceMarkdownImages";

describe("workspaceMarkdownImages", () => {
  afterEach(() => {
    clearWorkspaceMarkdownImageLoadCache();
  });

  it("remembers a loaded source until it fails", () => {
    rememberWorkspaceMarkdownImageLoad("https://files.example/a.png");

    expect(workspaceMarkdownImageIsLoaded("https://files.example/a.png")).toBe(true);

    forgetWorkspaceMarkdownImageLoad("https://files.example/a.png");

    expect(workspaceMarkdownImageIsLoaded("https://files.example/a.png")).toBe(false);
  });

  it("does not treat a near-expiry signed URL as loaded", () => {
    const expiring = `https://files.example/a.png?expires=${Math.floor(Date.now() / 1000) + 10}&sig=old`;
    rememberWorkspaceMarkdownImageLoad(expiring);

    expect(workspaceMarkdownImageIsLoaded(expiring)).toBe(false);
  });

  it("evicts the oldest source after 200 loads", () => {
    for (let index = 0; index < 200; index += 1) {
      rememberWorkspaceMarkdownImageLoad(`https://files.example/${index}.png`);
    }
    rememberWorkspaceMarkdownImageLoad("https://files.example/newest.png");

    expect(workspaceMarkdownImageIsLoaded("https://files.example/0.png")).toBe(false);
    expect(workspaceMarkdownImageIsLoaded("https://files.example/1.png")).toBe(true);
    expect(workspaceMarkdownImageIsLoaded("https://files.example/newest.png")).toBe(true);
  });
});
