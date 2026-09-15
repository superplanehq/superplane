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

  it("blocks when GitHub has no hosted app", () => {
    expect(
      githubAppAvailabilityFromCatalog({
        isSuccess: true,
        isError: false,
        githubDefinition: { name: "github", hostedAppInstall: false },
      }),
    ).toEqual({ resolved: true, available: false, failed: false });
  });

  it("allows setup when GitHub has a hosted app", () => {
    expect(
      githubAppAvailabilityFromCatalog({
        isSuccess: true,
        isError: false,
        githubDefinition: { name: "github", hostedAppInstall: true },
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
