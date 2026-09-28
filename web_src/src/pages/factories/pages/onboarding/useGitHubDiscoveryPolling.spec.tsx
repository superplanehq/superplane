import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { useRecheckGitHubInstallRequest } from "@/hooks/useRecheckGitHubInstallRequest";

import { useGitHubDiscoveryPolling } from "./useGitHubDiscoveryPolling";

const followBrowserAction = vi.hoisted(() => vi.fn(() => true));
vi.mock("@/lib/browserAction", () => ({ followBrowserAction }));
vi.mock("@/hooks/useRecheckGitHubInstallRequest", () => ({
  INSTALLATION_DISCOVERY_RECHECK_INTERVAL_MS: 1_000,
  INSTALL_REQUEST_RECHECK_INTERVAL_MS: 5_000,
  useRecheckGitHubInstallRequest: vi.fn(() => ({ failed: false, retry: vi.fn() })),
}));

const baseArgs = {
  organizationId: "org-1",
  integrationId: "int-1",
  enabled: true,
  discoveryActive: true,
  connectScreenOpen: true,
  pickerOpen: true,
  installRequested: false,
};

describe("useGitHubDiscoveryPolling", () => {
  beforeEach(() => {
    vi.mocked(useRecheckGitHubInstallRequest).mockClear();
    followBrowserAction.mockClear();
  });

  it("polls active discovery every second", () => {
    renderHook(() => useGitHubDiscoveryPolling(baseArgs));

    expect(vi.mocked(useRecheckGitHubInstallRequest)).toHaveBeenCalledWith("org-1", "int-1", true, 1_000);
  });

  it("follows one completed empty-discovery action once", () => {
    const browserAction = {
      id: "int-1",
      action: { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=csrf" },
    };
    const { rerender } = renderHook(
      ({ discoveryActive }) => useGitHubDiscoveryPolling({ ...baseArgs, discoveryActive, browserAction }),
      { initialProps: { discoveryActive: true } },
    );

    expect(followBrowserAction).not.toHaveBeenCalled();
    rerender({ discoveryActive: false });
    rerender({ discoveryActive: false });

    expect(followBrowserAction).toHaveBeenCalledTimes(1);
  });

  it("does not follow a stale action before discovery is observed", () => {
    const browserAction = {
      id: "int-1",
      action: { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=csrf" },
    };

    renderHook(() => useGitHubDiscoveryPolling({ ...baseArgs, discoveryActive: false, browserAction }));

    expect(followBrowserAction).not.toHaveBeenCalled();
  });
});
