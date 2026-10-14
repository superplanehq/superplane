import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type * as ReactRouterModule from "react-router";
import { unmockedPackage, unmockedSrc } from "@/test/unmockedModule";

import type * as StartDirectJiraConnectModule from "@/lib/startDirectJiraConnect";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";

import {
  selectReadyIntegrationInstance,
  useHostedJiraConnect,
  useIntegrationConnectDialog,
} from "./useIntegrationConnectDialog";

const startDirectJiraConnect = vi.hoisted(() => vi.fn().mockResolvedValue(true));

vi.mock("@/lib/startDirectJiraConnect", () => {
  const actual = unmockedSrc<typeof StartDirectJiraConnectModule>("lib/startDirectJiraConnect");
  return { ...actual, startDirectJiraConnect };
});

vi.mock("react-router", () => {
  const actual = unmockedPackage<typeof ReactRouterModule>("react-router/dist/development/index.js");
  return { ...actual, useNavigate: () => vi.fn() };
});

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "user-1" } }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: vi.fn(() => ({ data: [], refetch: vi.fn(), isLoading: false })),
  useAvailableIntegrations: vi.fn(() => ({ data: [] })),
  useCreateIntegration: vi.fn(() => ({ mutateAsync: vi.fn() })),
}));

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

describe("useIntegrationConnectDialog", () => {
  it("does not start navigation when Connect selects a saved connection", async () => {
    const onSelectionsChange = vi.fn();
    vi.mocked(useConnectedIntegrations).mockReturnValue({
      data: [
        {
          metadata: { id: "github-1", name: "github", integrationName: "github" },
          status: { state: "ready" },
        },
      ],
      refetch: vi.fn(),
      isLoading: false,
    } as unknown as ReturnType<typeof useConnectedIntegrations>);
    vi.mocked(useAvailableIntegrations).mockReturnValue({
      data: [],
    } as unknown as ReturnType<typeof useAvailableIntegrations>);
    vi.mocked(useCreateIntegration).mockReturnValue({
      mutateAsync: vi.fn(),
    } as unknown as ReturnType<typeof useCreateIntegration>);

    const { result } = renderHook(() =>
      useIntegrationConnectDialog({
        organizationId: "org-1",
        integrationNames: ["github"],
        selections: {},
        onSelectionsChange,
        manualSelectionNames: ["github"],
      }),
    );

    let navigationStarted = true;
    await act(async () => {
      navigationStarted = await result.current.requestConnect("github");
    });

    expect(navigationStarted).toBe(false);
    expect(onSelectionsChange).toHaveBeenCalledWith({
      github: { id: "github-1", name: "github", ready: true },
    });
  });
});
