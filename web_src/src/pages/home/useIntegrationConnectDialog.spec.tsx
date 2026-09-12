import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { OrganizationsCreateIntegrationResponse } from "@/api-client";
import type * as StartDirectGitHubConnectModule from "@/lib/startDirectGitHubConnect";
import type * as ReactRouterModule from "react-router";

import { useHostedGitHubConnect } from "./useIntegrationConnectDialog";

const startDirectGitHubConnect = vi.hoisted(() => vi.fn().mockResolvedValue(true));

vi.mock("@/lib/startDirectGitHubConnect", async (importOriginal) => {
  const actual = await importOriginal<typeof StartDirectGitHubConnectModule>();
  return { ...actual, startDirectGitHubConnect };
});

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof ReactRouterModule>();
  return { ...actual, useNavigate: () => vi.fn() };
});

describe("useHostedGitHubConnect", () => {
  beforeEach(() => {
    startDirectGitHubConnect.mockClear();
  });

  it("keeps a queued connect pending until the current user loads", async () => {
    const createIntegration = vi.fn<() => Promise<{ data: OrganizationsCreateIntegrationResponse }>>();
    const existingIntegrationNames = new Set<string>();
    const { result, rerender } = renderHook(
      ({ currentUserId, currentUserResolved }: { currentUserId?: string; currentUserResolved: boolean }) =>
        useHostedGitHubConnect({
          organizationId: "org-1",
          returnTo: "/org-1/workspaces/app/setup?step=vcs",
          connected: [],
          existingIntegrationNames,
          currentUserId,
          currentUserResolved,
          createIntegration,
        }),
      { initialProps: { currentUserId: undefined as string | undefined, currentUserResolved: false } },
    );

    const navigation = result.current();
    let settled = false;
    void navigation.then(() => {
      settled = true;
    });
    await act(async () => Promise.resolve());

    expect(settled).toBe(false);
    expect(startDirectGitHubConnect).not.toHaveBeenCalled();

    rerender({ currentUserId: "user-1", currentUserResolved: true });
    await expect(navigation).resolves.toBe(true);
    expect(startDirectGitHubConnect).toHaveBeenCalledWith(
      expect.objectContaining({ currentUserId: "user-1", organizationId: "org-1" }),
    );
  });

  it("releases a queued connect when the current user cannot load", async () => {
    const { result, rerender } = renderHook(
      ({ currentUserResolved }: { currentUserResolved: boolean }) =>
        useHostedGitHubConnect({
          organizationId: "org-1",
          connected: [],
          existingIntegrationNames: new Set(),
          currentUserResolved,
          createIntegration: vi.fn(),
        }),
      { initialProps: { currentUserResolved: false } },
    );

    const navigation = result.current();
    rerender({ currentUserResolved: true });

    await expect(navigation).resolves.toBe(false);
    expect(startDirectGitHubConnect).not.toHaveBeenCalled();
  });
});
