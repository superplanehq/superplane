import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import type { OrganizationsIntegration } from "@/api-client";

import { pendingGitHubInstallRequestId, useRecheckGitHubInstallRequest } from "./useRecheckGitHubInstallRequest";

const updateIntegration = vi.hoisted(() => vi.fn().mockResolvedValue({}));
vi.mock("@/api-client/sdk.gen", () => ({
  organizationsUpdateIntegration: updateIntegration,
}));

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>;
}

const waitingInstance: OrganizationsIntegration = {
  metadata: { id: "int-1", integrationName: "github" },
  status: { state: "pending", metadata: { installRequested: true, installRequestedAccount: "acme" } },
};

const readyInstance: OrganizationsIntegration = {
  metadata: { id: "int-2", integrationName: "github" },
  status: { state: "ready", metadata: { owner: "acme" } },
};

describe("pendingGitHubInstallRequestId", () => {
  it("finds the pending GitHub connection with an install request", () => {
    expect(pendingGitHubInstallRequestId([readyInstance, waitingInstance], "user-1")).toBe("int-1");
    expect(pendingGitHubInstallRequestId([readyInstance], "user-1")).toBeUndefined();
  });

  it("selects the current user's request instead of another member's request", () => {
    const theirs: OrganizationsIntegration = {
      ...waitingInstance,
      metadata: { ...waitingInstance.metadata, id: "int-theirs" },
      status: { ...waitingInstance.status, metadata: { installRequested: true, startedByUserID: "user-2" } },
    };
    const mine: OrganizationsIntegration = {
      ...waitingInstance,
      metadata: { ...waitingInstance.metadata, id: "int-mine" },
      status: { ...waitingInstance.status, metadata: { installRequested: true, startedByUserID: "user-1" } },
    };

    expect(pendingGitHubInstallRequestId([theirs, mine], "user-1")).toBe("int-mine");
  });

  it("finds an install request on a ready connection", () => {
    const readyWithRequest: OrganizationsIntegration = {
      ...readyInstance,
      status: {
        ...readyInstance.status,
        metadata: { installRequested: true, startedByUserID: "user-1", installRequestedAccount: "octo" },
      },
    };

    expect(pendingGitHubInstallRequestId([readyWithRequest], "user-1")).toBe("int-2");
  });
});

describe("useRecheckGitHubInstallRequest", () => {
  afterEach(() => {
    vi.useRealTimers();
    updateIntegration.mockReset().mockResolvedValue({});
  });

  it("rechecks the pending install request on page access", async () => {
    renderHook(() => useRecheckGitHubInstallRequest("org-1", "int-1"), { wrapper });

    await waitFor(() => expect(updateIntegration).toHaveBeenCalledTimes(1));
    const args = updateIntegration.mock.calls[0][0] as { path: { id: string; integrationId: string } };
    expect(args.path).toEqual({ id: "org-1", integrationId: "int-1" });
  });

  it("does nothing without a pending install request", () => {
    renderHook(() => useRecheckGitHubInstallRequest("org-1", undefined), { wrapper });

    expect(updateIntegration).not.toHaveBeenCalled();
  });

  it("does nothing when polling is disabled", () => {
    renderHook(() => useRecheckGitHubInstallRequest("org-1", "int-1", false), { wrapper });

    expect(updateIntegration).not.toHaveBeenCalled();
  });

  it("waits for each recheck before scheduling the next one and stops on cleanup", async () => {
    vi.useFakeTimers();
    let finishFirst!: () => void;
    updateIntegration.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishFirst = () => resolve({});
        }),
    );

    const { unmount } = renderHook(() => useRecheckGitHubInstallRequest("org-1", "int-1"), { wrapper });
    await act(async () => Promise.resolve());
    expect(updateIntegration).toHaveBeenCalledTimes(1);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(updateIntegration).toHaveBeenCalledTimes(1);

    await act(async () => {
      finishFirst();
      await Promise.resolve();
    });
    await act(async () => vi.advanceTimersByTimeAsync(4_999));
    expect(updateIntegration).toHaveBeenCalledTimes(1);
    await act(async () => vi.advanceTimersByTimeAsync(1));
    expect(updateIntegration).toHaveBeenCalledTimes(2);

    unmount();
    await act(async () => vi.advanceTimersByTimeAsync(5_000));
    expect(updateIntegration).toHaveBeenCalledTimes(2);
  });
});
