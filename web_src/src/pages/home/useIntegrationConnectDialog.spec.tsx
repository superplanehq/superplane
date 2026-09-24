import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { OrganizationsCreateIntegrationResponse } from "@/api-client";
import type * as StartDirectGitHubConnectModule from "@/lib/startDirectGitHubConnect";
import type * as ReactRouterModule from "react-router";
import { unmockedPackage, unmockedSrc } from "@/test/unmockedModule";

import type * as StartDirectJiraConnectModule from "@/lib/startDirectJiraConnect";

import {
  selectReadyIntegrationInstance,
  useHostedGitHubConnect,
  useHostedJiraConnect,
} from "./useIntegrationConnectDialog";

const startDirectGitHubConnect = vi.hoisted(() => vi.fn().mockResolvedValue(true));
const startDirectJiraConnect = vi.hoisted(() => vi.fn().mockResolvedValue(true));

vi.mock("@/lib/startDirectGitHubConnect", () => {
  const actual = unmockedSrc<typeof StartDirectGitHubConnectModule>("lib/startDirectGitHubConnect");
  return { ...actual, startDirectGitHubConnect };
});

vi.mock("@/lib/startDirectJiraConnect", () => {
  const actual = unmockedSrc<typeof StartDirectJiraConnectModule>("lib/startDirectJiraConnect");
  return { ...actual, startDirectJiraConnect };
});

vi.mock("react-router", () => {
  const actual = unmockedPackage<typeof ReactRouterModule>("react-router/dist/development/index.js");
  return { ...actual, useNavigate: () => vi.fn() };
});

describe("useHostedGitHubConnect", () => {
  beforeEach(() => {
    startDirectGitHubConnect.mockClear();
    startDirectJiraConnect.mockClear();
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

describe("useHostedJiraConnect", () => {
  beforeEach(() => {
    startDirectJiraConnect.mockClear();
  });

  it("passes forceNew through so Connect new can create another connection", async () => {
    const connected = [{ metadata: { integrationName: "jira", id: "jira-1" }, status: { state: "ready" } }];
    const { result } = renderHook(() =>
      useHostedJiraConnect({
        organizationId: "org-1",
        connected,
        existingIntegrationNames: new Set(["jira"]),
        createIntegration: vi.fn(),
      }),
    );

    await act(async () => {
      await result.current(true);
    });

    expect(startDirectJiraConnect).toHaveBeenCalledWith(expect.objectContaining({ forceNew: true, connected }));
  });

  it("reuses existing connections when forceNew is omitted", async () => {
    const { result } = renderHook(() =>
      useHostedJiraConnect({
        organizationId: "org-1",
        connected: [],
        existingIntegrationNames: new Set(),
        createIntegration: vi.fn(),
      }),
    );

    await act(async () => {
      await result.current();
    });

    expect(startDirectJiraConnect).toHaveBeenCalledWith(expect.objectContaining({ forceNew: false }));
  });
});

describe("selectReadyIntegrationInstance", () => {
  it("selects an existing ready provider when no instance id is specified", () => {
    const connected = [
      {
        metadata: { id: "openrouter-1", name: "openrouter", integrationName: "openrouter" },
        status: { state: "ready" },
      },
    ];

    expect(selectReadyIntegrationInstance(connected, {}, "openrouter")).toEqual({
      openrouter: { id: "openrouter-1", name: "openrouter", ready: true },
    });
  });
});
