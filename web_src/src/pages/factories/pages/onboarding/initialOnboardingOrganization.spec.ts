import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import {
  isOrganizationIdentityTaken,
  nameOrganizationFromGitHubOwner,
  organizationIdentityFromOwner,
  shouldNameOrganizationFromGitHub,
} from "./initialOnboardingOrganization";

function factoryWithOnboarding(onboarding: FactoriesFactory["onboarding"]): FactoriesFactory {
  return { onboarding } as FactoriesFactory;
}

describe("shouldNameOrganizationFromGitHub", () => {
  it("names only an organization from the initial account onboarding workspace", () => {
    expect(shouldNameOrganizationFromGitHub(factoryWithOnboarding({ initial: true }), true)).toBe(true);
    expect(shouldNameOrganizationFromGitHub(factoryWithOnboarding({}), true)).toBe(false);
  });

  it("retries naming after the GitHub connection was saved", () => {
    const factory = factoryWithOnboarding({ initial: true, vcsIntegrationId: "integration-id" });
    expect(shouldNameOrganizationFromGitHub(factory, true)).toBe(true);
  });

  it("does not rename when the user selects an existing connection", () => {
    expect(shouldNameOrganizationFromGitHub(factoryWithOnboarding({ initial: true }), false)).toBe(false);
  });
});

describe("organizationIdentityFromOwner", () => {
  it("uses the GitHub owner for the name and slug", () => {
    expect(organizationIdentityFromOwner("Acme Org")).toEqual({ name: "Acme Org", slug: "acme-org" });
  });

  it("appends a random suffix when the owner is already taken", () => {
    expect(organizationIdentityFromOwner("Acme Org", "1ajioa")).toEqual({
      name: "Acme Org-1ajioa",
      slug: "acme-org-1ajioa",
    });
  });
});

describe("isOrganizationIdentityTaken", () => {
  it("detects a taken slug or name", () => {
    expect(isOrganizationIdentityTaken("organization slug is already in use")).toBe(true);
    expect(isOrganizationIdentityTaken("name already used")).toBe(true);
    expect(isOrganizationIdentityTaken("Could not save the organization")).toBe(false);
  });
});

describe("nameOrganizationFromGitHubOwner", () => {
  it("retries with a suffix when the owner slug is already in use", async () => {
    const update = vi
      .fn()
      .mockRejectedValueOnce({ response: { data: { message: "organization slug is already in use" } } })
      .mockResolvedValueOnce("acme-1ajioa");

    await expect(
      nameOrganizationFromGitHubOwner({
        owner: "acme",
        currentSlug: "dev-user",
        update,
        randomSuffix: () => "1ajioa",
      }),
    ).resolves.toBe("acme-1ajioa");

    expect(update).toHaveBeenNthCalledWith(1, { name: "acme", slug: "acme" });
    expect(update).toHaveBeenNthCalledWith(2, { name: "acme-1ajioa", slug: "acme-1ajioa" });
  });
});
