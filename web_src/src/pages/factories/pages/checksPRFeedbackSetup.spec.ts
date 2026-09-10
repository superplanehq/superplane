import {
  catalogStatusCheckNames,
  suggestedIntegrationsForChecks,
  suggestIntegrationFromCheckURL,
} from "./checksPRFeedbackSetup";

describe("checksPRFeedbackSetup helpers", () => {
  it("collects catalog check names", () => {
    expect(catalogStatusCheckNames([{ name: "lint" }, { name: "unit" }, { name: "  " }, {}])).toEqual(["lint", "unit"]);
  });

  it("suggests integrations from check urls", () => {
    expect(suggestIntegrationFromCheckURL("https://acme.semaphoreci.com/workflows/abc")).toBe("semaphore");
    expect(suggestIntegrationFromCheckURL("https://app.circleci.com/pipelines/github/acme/api/12")).toBe("circleci");
    expect(suggestIntegrationFromCheckURL("https://github.com/acme/api/actions/runs/99")).toBe("");
    expect(suggestIntegrationFromCheckURL("")).toBe("");
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
});
