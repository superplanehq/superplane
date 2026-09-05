import { describe, expect, it } from "vitest";

import type { OrganizationsIntegration } from "@/api-client";

import { findBrokenIntegrations, repairActionLabel } from "./brokenIntegrations";

function instance(
  overrides: Partial<{
    id: string;
    name: string;
    integrationName: string;
    state: "ready" | "pending" | "error";
    stateDescription: string;
    currentStep: object;
  }> = {},
): OrganizationsIntegration {
  return {
    metadata: {
      id: overrides.id ?? "integration-1",
      name: overrides.name ?? "github-main",
      integrationName: overrides.integrationName ?? "github",
    },
    status: {
      state: overrides.state ?? "ready",
      stateDescription: overrides.stateDescription,
      setupState: overrides.currentStep ? { currentStep: overrides.currentStep } : undefined,
    },
  } as OrganizationsIntegration;
}

describe("repairActionLabel", () => {
  it("suggests reinstalling the app when the description mentions an uninstall", () => {
    expect(repairActionLabel("App was uninstalled")).toBe("Reinstall app");
  });

  it("suggests replacing the key when the description mentions an expired credential", () => {
    expect(repairActionLabel("API key expired")).toBe("Replace key");
  });

  it("suggests reconnecting when an OAuth refresh token is missing", () => {
    expect(repairActionLabel("No refresh token was returned; reconnect with the offline_access scope")).toBe(
      "Reconnect",
    );
  });

  it("suggests reconnecting when the description asks to re-authorize", () => {
    expect(repairActionLabel("Authorization revoked, please reconnect the account")).toBe("Reconnect");
  });

  it("suggests reconnecting when an OAuth callback fails to persist the access token", () => {
    // Jira emits this on the OAuth callback path; recovery is a re-authorize.
    expect(repairActionLabel("failed to store access token: write timeout")).toBe("Reconnect");
    // Linear (and GitLab's OAuth path) emit this variant.
    expect(repairActionLabel("failed to save access token: write timeout")).toBe("Reconnect");
  });

  it("suggests reconnecting when an OAuth callback fails to schedule the token refresh", () => {
    // Jira emits this on the OAuth callback path; the reversed word order
    // ("token refresh") must not fall through to the "Replace key" hint.
    expect(repairActionLabel("failed to schedule token refresh: queue unavailable")).toBe("Reconnect");
  });

  it("still suggests replacing the key when a pasted access token is missing", () => {
    // GitLab's personal-access-token path emits this; recovery is a new key.
    expect(repairActionLabel("access token is required")).toBe("Replace key");
  });

  it("falls back to reconnect for an unrecognized description", () => {
    expect(repairActionLabel("Something went wrong")).toBe("Reconnect");
  });

  it("falls back to reconnect when there is no description", () => {
    expect(repairActionLabel(undefined)).toBe("Reconnect");
  });
});

describe("findBrokenIntegrations", () => {
  it("returns nothing when every integration is ready", () => {
    expect(findBrokenIntegrations([instance({ state: "ready" })])).toEqual([]);
  });

  it("flags an integration in the error state with its repair step", () => {
    const broken = findBrokenIntegrations([
      instance({ id: "gh-1", integrationName: "github", state: "error", stateDescription: "App was uninstalled" }),
    ]);

    expect(broken).toEqual([
      {
        id: "gh-1",
        name: "github-main",
        integrationName: "github",
        reason: "error",
        description: "App was uninstalled",
        actionLabel: "Reinstall app",
      },
    ]);
  });

  it("flags a pending integration with no active setup step as incomplete", () => {
    const broken = findBrokenIntegrations([instance({ id: "op-1", integrationName: "openai", state: "pending" })]);

    expect(broken).toEqual([
      {
        id: "op-1",
        name: "github-main",
        integrationName: "openai",
        reason: "incomplete",
        description: "Setup is not finished.",
        actionLabel: "Finish setup",
      },
    ]);
  });

  it("does not flag a pending integration that is still mid-wizard", () => {
    const broken = findBrokenIntegrations([
      instance({ state: "pending", currentStep: { type: "INPUTS", name: "step" } }),
    ]);

    expect(broken).toEqual([]);
  });

  it("skips integrations missing required metadata", () => {
    const incomplete = {
      metadata: { name: "no-id" },
      status: { state: "error" },
    } as OrganizationsIntegration;

    expect(findBrokenIntegrations([incomplete])).toEqual([]);
  });
});
