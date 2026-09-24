import { describe, expect, it } from "bun:test";

import { githubAppAvailabilityFromCatalog } from "./useGithubAppAvailability";

describe("githubAppAvailabilityFromCatalog", () => {
  it("waits while the catalog is still loading", () => {
    expect(githubAppAvailabilityFromCatalog({ isSuccess: false, isError: false })).toEqual({
      resolved: false,
      available: false,
      failed: false,
    });
  });

  it("blocks when the process has no GitHub App", () => {
    expect(
      githubAppAvailabilityFromCatalog({
        isSuccess: true,
        isError: false,
        githubAppConfigured: false,
      }),
    ).toEqual({ resolved: true, available: false, failed: false });
  });

  it("allows setup when the process holds a GitHub App", () => {
    expect(
      githubAppAvailabilityFromCatalog({
        isSuccess: true,
        isError: false,
        githubAppConfigured: true,
      }),
    ).toEqual({ resolved: true, available: true, failed: false });
  });

  it("allows setup when the process holds a GitHub App even if hosted install is false", () => {
    expect(
      githubAppAvailabilityFromCatalog({
        isSuccess: true,
        isError: false,
        githubAppConfigured: true,
        githubDefinition: { name: "github", hostedAppInstall: false },
      }),
    ).toEqual({ resolved: true, available: true, failed: false });
  });

  it("settles as failed when the catalog request fails", () => {
    expect(githubAppAvailabilityFromCatalog({ isSuccess: false, isError: true })).toEqual({
      resolved: true,
      available: false,
      failed: true,
    });
  });
});
