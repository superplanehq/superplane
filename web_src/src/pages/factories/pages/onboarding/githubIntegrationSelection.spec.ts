import { describe, expect, it } from "bun:test";

import { githubIntegrationSelection, selectionsWithGitHubInstallation } from "./githubIntegrationSelection";

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
