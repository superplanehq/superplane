import { describe, expect, it } from "bun:test";

import {
  githubIntegrationSelection,
  selectionsWithGitHubInstallation,
  selectionsWithSavedVcsInstallation,
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

describe("selectionsWithSavedVcsInstallation", () => {
  it("does not mark a saved integration ready when the name is still the id", () => {
    const selections = { bitbucket: { id: "int-9", name: "int-9", ready: false } };

    expect(
      selectionsWithSavedVcsInstallation({ vcsIntegrationId: "int-9", vcsProvider: "bitbucket" }, selections, "int-9"),
    ).toBe(selections);
    expect(
      selectionsWithSavedVcsInstallation({ vcsIntegrationId: "int-9", vcsProvider: "bitbucket" }, selections, "  "),
    ).toBe(selections);
  });

  it("marks Bitbucket ready only after the installation name is known", () => {
    expect(
      selectionsWithSavedVcsInstallation({ vcsIntegrationId: "int-9", vcsProvider: "bitbucket" }, {}, "bitbucket-acme"),
    ).toEqual({
      bitbucket: { id: "int-9", name: "bitbucket-acme", ready: true },
    });
  });

  it("replaces a ready placeholder name before install", () => {
    expect(
      selectionsWithSavedVcsInstallation(
        { vcsIntegrationId: "int-1", vcsProvider: "github" },
        { github: { id: "int-1", name: "int-1", ready: true } },
        "github-acme",
      ).github,
    ).toEqual({ id: "int-1", name: "github-acme", ready: true });
  });

  it("keeps an already named selection", () => {
    const selections = { github: { id: "int-1", name: "github-acme", ready: true } };
    expect(
      selectionsWithSavedVcsInstallation(
        { vcsIntegrationId: "int-1", vcsProvider: "github" },
        selections,
        "github-acme",
      ),
    ).toBe(selections);
  });

  it("leaves selections unchanged when nothing was saved", () => {
    const selections = { claude: { id: "int-2", name: "claude", ready: true } };
    expect(selectionsWithSavedVcsInstallation(undefined, selections, "github-acme")).toBe(selections);
    expect(selectionsWithSavedVcsInstallation({ vcsProvider: "github" }, selections, "github-acme")).toBe(selections);
  });
});
