import { describe, expect, it } from "bun:test";

import {
  catalogStatusCheckNames,
  checksToolsAccess,
  isGitHubActionsCheckURL,
  selectedChecksUseGitHubActions,
  suggestedIntegrationsForChecks,
  suggestIntegrationFromCheckURL,
} from "./checksPRFeedbackSetup";

describe("checksPRFeedbackSetup helpers", () => {
  it("collects catalog check names", () => {
    expect(catalogStatusCheckNames([{ name: "lint" }, { name: "unit" }, { name: "  " }, {}])).toEqual(["lint", "unit"]);
  });

  it("prefers resource ids with name fallback", () => {
    expect(
      catalogStatusCheckNames([{ id: "build-a", name: "Build A" }, { name: "lint" }, { id: "  ", name: "  " }, {}]),
    ).toEqual(["build-a", "lint"]);
  });

  it("suggests integrations from check urls", () => {
    expect(suggestIntegrationFromCheckURL("https://acme.semaphoreci.com/workflows/abc")).toBe("semaphore");
    expect(suggestIntegrationFromCheckURL("https://app.circleci.com/pipelines/github/acme/api/12")).toBe("circleci");
    expect(suggestIntegrationFromCheckURL("https://github.com/acme/api/actions/runs/99")).toBe("");
    expect(suggestIntegrationFromCheckURL("")).toBe("");
  });

  it("detects GitHub Actions check urls", () => {
    expect(isGitHubActionsCheckURL("https://github.com/acme/api/actions/runs/99")).toBe(true);
    expect(isGitHubActionsCheckURL("https://github.com/acme/api/actions/runs/99/job/12")).toBe(true);
    expect(isGitHubActionsCheckURL("https://api.github.com/repos/acme/api/actions/runs/99")).toBe(true);
    expect(isGitHubActionsCheckURL("https://github.com/acme/api/pull/12")).toBe(false);
    expect(isGitHubActionsCheckURL("https://app.circleci.com/pipelines/1")).toBe(false);
    expect(isGitHubActionsCheckURL("")).toBe(false);
  });

  it("keeps only common status-check integrations", () => {
    expect(
      suggestedIntegrationsForChecks(
        [
          { name: "lint", url: "https://acme.semaphoreci.com/workflows/1" },
          { name: "e2e", url: "https://app.circleci.com/pipelines/1" },
          { name: "actions", url: "https://github.com/acme/api/actions/runs/1" },
        ],
        ["lint", "e2e", "actions"],
      ),
    ).toEqual(["semaphore", "circleci"]);
    expect(
      suggestedIntegrationsForChecks([{ name: "lint", url: "https://ci.example.com/job/lint" }], ["lint"]),
    ).toEqual([]);
  });

  it("detects when selected checks use GitHub Actions", () => {
    const catalog = [
      { name: "build", url: "https://github.com/acme/api/actions/runs/1" },
      { name: "e2e", url: "https://app.circleci.com/pipelines/1" },
    ];
    expect(selectedChecksUseGitHubActions(catalog, ["build"])).toBe(true);
    expect(selectedChecksUseGitHubActions(catalog, ["e2e"])).toBe(false);
    expect(selectedChecksUseGitHubActions(catalog, ["build", "e2e"])).toBe(true);
  });

  it("matches suggestions and GitHub Actions by resource id", () => {
    const catalog = [
      { id: "build-a", name: "Build A", url: "https://app.circleci.com/pipelines/1" },
      { id: "build-b", name: "Build B", url: "https://github.com/acme/api/actions/runs/1" },
    ];
    expect(suggestedIntegrationsForChecks(catalog, ["build-a"])).toEqual(["circleci"]);
    expect(suggestedIntegrationsForChecks(catalog, ["Build A"])).toEqual([]);
    expect(selectedChecksUseGitHubActions(catalog, ["build-b"])).toBe(true);
    expect(selectedChecksUseGitHubActions(catalog, ["Build B"])).toBe(false);
  });

  it("keeps GitHub selection behavior when id and name match", () => {
    const catalog = [
      { id: "lint", name: "lint", url: "https://acme.semaphoreci.com/workflows/1" },
      { id: "e2e", name: "e2e", url: "https://app.circleci.com/pipelines/1" },
    ];
    expect(suggestedIntegrationsForChecks(catalog, ["lint", "e2e"])).toEqual(["semaphore", "circleci"]);
    expect(selectedChecksUseGitHubActions(catalog, ["lint"])).toBe(false);
  });

  it("classifies tools-step access from suggestions and GitHub Actions", () => {
    expect(checksToolsAccess(["circleci"], true)).toBe("suggested");
    expect(checksToolsAccess([], true)).toBe("github-actions");
    expect(checksToolsAccess([], false)).toBe("none");
  });
});
