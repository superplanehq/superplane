import { describe, expect, it } from "bun:test";

import {
  githubIntegrationSelection,
  selectionsWithGitHubInstallation,
  selectionsWithSavedVcsReady,
} from "./githubIntegrationSelection";

describe("githubIntegrationSelection", () => {
  it("stores the installation name, not the product label", () => {
    expect(githubIntegrationSelection("  int-1  ", "  github-acme  ")).toEqual({
      id: "int-1",
      name: "github-acme",
      ready: true,
    });
  });

  it("refuses a blank installation name", () => {
    expect(() => githubIntegrationSelection("int-1", "  ")).toThrow("GitHub integration name is missing");
  });
});

describe("selectionsWithGitHubInstallation", () => {
  it("replaces a product-label GitHub selection before canvas install", () => {
    const selections = selectionsWithGitHubInstallation(
      {
        github: { id: "int-1", name: "GitHub", ready: true },
        claude: { id: "int-2", name: "claude", ready: true },
      },
      "github-acme",
    );

    expect(selections.github).toEqual({ id: "int-1", name: "github-acme", ready: true });
    expect(selections.claude).toEqual({ id: "int-2", name: "claude", ready: true });
  });

  it("leaves selections unchanged when GitHub is not selected", () => {
    const selections = { claude: { id: "int-2", name: "claude", ready: true } };
    expect(selectionsWithGitHubInstallation(selections, "github-acme")).toBe(selections);
  });
});

describe("selectionsWithSavedVcsReady", () => {
  it("restores a saved GitHub integration after an OAuth reload", () => {
    const selections = selectionsWithSavedVcsReady(
      { vcsIntegrationId: "int-1", vcsProvider: "github" },
      { github: { id: "int-1", name: "int-1", ready: false } },
    );

    expect(selections.github).toEqual({ id: "int-1", name: "int-1", ready: true });
  });

  it("restores a saved Bitbucket integration after an OAuth reload", () => {
    const selections = selectionsWithSavedVcsReady({ vcsIntegrationId: "int-9", vcsProvider: "bitbucket" }, {});

    expect(selections.bitbucket).toEqual({ id: "int-9", name: "int-9", ready: true });
  });

  it("keeps an already ready selection untouched", () => {
    const selections = { github: { id: "int-1", name: "github-acme", ready: true } };
    expect(selectionsWithSavedVcsReady({ vcsIntegrationId: "int-1", vcsProvider: "github" }, selections)).toBe(
      selections,
    );
  });

  it("leaves selections unchanged when nothing was saved", () => {
    const selections = { claude: { id: "int-2", name: "claude", ready: true } };
    expect(selectionsWithSavedVcsReady(undefined, selections)).toBe(selections);
    expect(selectionsWithSavedVcsReady({ vcsProvider: "github" }, selections)).toBe(selections);
  });
});
