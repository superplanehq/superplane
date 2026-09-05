import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import {
  isOrganizationIdentityTaken,
  nameOrganizationFromGitHubOwner,
  organizationIdentityFromOwner,
  organizationSlugMatchesOwner,
  shouldNameOrganizationFromGitHub,
} from "./initialOnboardingOrganization";

function factoryWithOnboarding(onboarding: FactoriesFactory["onboarding"]): FactoriesFactory {
  return { onboarding } as FactoriesFactory;
}

describe("shouldNameOrganizationFromGitHub", () => {
  it("names only an organization from the initial account onboarding workspace", () => {
    expect(shouldNameOrganizationFromGitHub(factoryWithOnboarding({ initial: true }))).toBe(true);
    expect(shouldNameOrganizationFromGitHub(factoryWithOnboarding({}))).toBe(false);
    expect(shouldNameOrganizationFromGitHub(null)).toBe(false);
  });

  it("retries naming after the GitHub connection was saved", () => {
    const factory = factoryWithOnboarding({ initial: true, vcsIntegrationId: "integration-id" });
    expect(shouldNameOrganizationFromGitHub(factory)).toBe(true);
  });
});

describe("organizationSlugMatchesOwner", () => {
  it("matches the plain owner slug and the suffixed slug", () => {
    expect(organizationSlugMatchesOwner("acme-org", "Acme Org")).toBe(true);
    expect(organizationSlugMatchesOwner("acme-org-1ajioa", "Acme Org")).toBe(true);
  });

  it("does not match a different slug or a non-suffix tail", () => {
    expect(organizationSlugMatchesOwner("dev-user", "Acme Org")).toBe(false);
    expect(organizationSlugMatchesOwner("acme-org-inc", "Acme Org")).toBe(false);
  });
});

describe("organizationIdentityFromOwner", () => {
  it("uses the GitHub owner for the name and slug", () => {
    expect(organizationIdentityFromOwner("Acme Org")).toEqual({ name: "Acme Org", slug: "acme-org" });
  });

  it("appends a random suffix to the slug only when the owner is already taken", () => {
    expect(organizationIdentityFromOwner("Acme Org", "1ajioa")).toEqual({
      name: "Acme Org",
      slug: "acme-org-1ajioa",
    });
  });
});

describe("isOrganizationIdentityTaken", () => {
  it("detects a taken slug or name", () => {
    expect(isOrganizationIdentityTaken("organization slug is already in use")).toBe(true);
    expect(isOrganizationIdentityTaken("organization name is already in use")).toBe(true);
    expect(isOrganizationIdentityTaken("name already used")).toBe(true);
    expect(isOrganizationIdentityTaken("invalid organization update")).toBe(true);
    expect(isOrganizationIdentityTaken("duplicate key value violates unique constraint")).toBe(true);
    expect(isOrganizationIdentityTaken("Could not save the organization")).toBe(false);
  });
});

describe("nameOrganizationFromGitHubOwner", () => {
  it("does not rename when the slug already derives from the owner", async () => {
    const update = vi.fn();

    await expect(nameOrganizationFromGitHubOwner({ owner: "acme", currentSlug: "acme", update })).resolves.toBe("acme");
    await expect(nameOrganizationFromGitHubOwner({ owner: "acme", currentSlug: "acme-1ajioa", update })).resolves.toBe(
      "acme-1ajioa",
    );

    expect(update).not.toHaveBeenCalled();
  });

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
    expect(update).toHaveBeenNthCalledWith(2, { name: "acme", slug: "acme-1ajioa" });
  });

  it("retries when the name conflict uses the update error text", async () => {
    const update = vi
      .fn()
      .mockRejectedValueOnce({ error: { message: "invalid organization update" } })
      .mockResolvedValueOnce("acme-1ajioa");

    await expect(
      nameOrganizationFromGitHubOwner({
        owner: "acme",
        currentSlug: "dev-user",
        update,
        randomSuffix: () => "1ajioa",
      }),
    ).resolves.toBe("acme-1ajioa");
  });

  it("keeps the plain owner as the name across every retry, only rotating the slug suffix", async () => {
    const update = vi
      .fn()
      .mockRejectedValueOnce({ response: { data: { message: "organization slug is already in use" } } })
      .mockRejectedValueOnce({ response: { data: { message: "organization slug is already in use" } } })
      .mockResolvedValueOnce("acme-8f3kd2");

    const suffixes = ["1ajioa", "8f3kd2"];
    let call = 0;

    await expect(
      nameOrganizationFromGitHubOwner({
        owner: "acme",
        currentSlug: "dev-user",
        update,
        randomSuffix: () => suffixes[call++],
      }),
    ).resolves.toBe("acme-8f3kd2");

    expect(update).toHaveBeenCalledTimes(3);
    expect(update).toHaveBeenNthCalledWith(1, { name: "acme", slug: "acme" });
    expect(update).toHaveBeenNthCalledWith(2, { name: "acme", slug: "acme-1ajioa" });
    expect(update).toHaveBeenNthCalledWith(3, { name: "acme", slug: "acme-8f3kd2" });

    const names = update.mock.calls.map(([identity]) => identity.name);
    expect(new Set(names).size).toBe(1);
    expect(names[0]).toBe("acme");
  });
});
