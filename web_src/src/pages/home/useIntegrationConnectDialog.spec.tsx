import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type * as ReactRouterModule from "react-router";
import { unmockedPackage, unmockedSrc } from "@/test/unmockedModule";

import type * as StartDirectJiraConnectModule from "@/lib/startDirectJiraConnect";

import { selectReadyIntegrationInstance, useHostedJiraConnect } from "./useIntegrationConnectDialog";

const startDirectJiraConnect = vi.hoisted(() => vi.fn().mockResolvedValue(true));

vi.mock("@/lib/startDirectJiraConnect", () => {
  const actual = unmockedSrc<typeof StartDirectJiraConnectModule>("lib/startDirectJiraConnect");
  return { ...actual, startDirectJiraConnect };
});

vi.mock("react-router", () => {
  const actual = unmockedPackage<typeof ReactRouterModule>("react-router/dist/development/index.js");
  return { ...actual, useNavigate: () => vi.fn() };
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
