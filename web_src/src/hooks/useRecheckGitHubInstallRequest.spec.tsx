import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

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
    expect(pendingGitHubInstallRequestId([readyInstance, waitingInstance])).toBe("int-1");
    expect(pendingGitHubInstallRequestId([readyInstance])).toBeUndefined();
  });
});

describe("useRecheckGitHubInstallRequest", () => {
  afterEach(() => {
    updateIntegration.mockClear();
  });

  it("rechecks the pending install request on page access", async () => {
    renderHook(() => useRecheckGitHubInstallRequest("org-1", [waitingInstance]), { wrapper });

    await waitFor(() => expect(updateIntegration).toHaveBeenCalledTimes(1));
    const args = updateIntegration.mock.calls[0][0] as { path: { id: string; integrationId: string } };
    expect(args.path).toEqual({ id: "org-1", integrationId: "int-1" });
  });

  it("does nothing without a pending install request", () => {
    renderHook(() => useRecheckGitHubInstallRequest("org-1", [readyInstance]), { wrapper });

    expect(updateIntegration).not.toHaveBeenCalled();
  });
});
