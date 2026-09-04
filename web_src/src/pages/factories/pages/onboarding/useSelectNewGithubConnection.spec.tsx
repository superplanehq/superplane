import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { OrganizationsIntegration } from "@/api-client";

import { useOnboardingGithubConnections } from "./useSelectNewGithubConnection";

const githubConnection: OrganizationsIntegration = {
  metadata: {
    id: "github-connection",
    name: "GitHub",
    integrationName: "github",
    createdAt: "2026-09-03T00:00:00Z",
  },
  status: {
    state: "ready",
    metadata: { owner: "forestileao" },
  },
};

describe("useOnboardingGithubConnections", () => {
  it("reports the newly connected GitHub integration to onboarding", async () => {
    const selectInstance = vi.fn();
    const onConnectionSelected = vi.fn();

    renderHook(() =>
      useOnboardingGithubConnections({
        integrationData: [{ name: "github", allInstances: [githubConnection], readyInstances: [githubConnection] }],
        openSection: "vcs",
        selectNewest: true,
        selections: {},
        selectInstance,
        onConnectionSelected,
      }),
    );

    await waitFor(() => expect(onConnectionSelected).toHaveBeenCalledWith(githubConnection));
    expect(selectInstance).toHaveBeenCalledWith("github", "github-connection");
  });
});
