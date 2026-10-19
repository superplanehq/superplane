import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { readBitbucketInstallWait, writeBitbucketInstallWait } from "./bitbucketInstallWait";
import { bitbucketInstallationAttemptMs } from "./bitbucketOnboardingPoll";
import {
  bitbucketOnboarding,
  bitbucketSetupElement,
  chooseBitbucket,
  renderSetup,
  resetBitbucketFirstRunHarness,
  resources,
} from "./FirstRunSetup.bitbucket.spec.fixtures";
import type { IntegrationId } from "./onboardingFixtures";

describe("FirstRunSetup Bitbucket", () => {
  beforeEach(() => {
    resetBitbucketFirstRunHarness();
  });

  it("offers GitHub and Bitbucket on the host screen and keeps GitLab disabled", async () => {
    const user = userEvent.setup();
    renderSetup(new Set());

    await user.click(screen.getByTestId("first-run-get-started"));

    expect(within(screen.getByTestId("first-run-host-github")).getByRole("button")).toBeEnabled();
    expect(within(screen.getByTestId("first-run-host-bitbucket")).getByRole("button")).toBeEnabled();
    expect(within(screen.getByTestId("first-run-host-gitlab")).getByRole("button")).toBeDisabled();
  });

  it("asks for a Bitbucket connection without a GitHub identity", async () => {
    const user = userEvent.setup();
    const requestConnect = vi.fn();
    renderSetup(new Set(), { requestConnect });

    await chooseBitbucket(user);
    await user.click(screen.getByTestId("first-run-connect-bitbucket"));

    expect(requestConnect).toHaveBeenCalledWith("bitbucket");
    expect(screen.queryByTestId("first-run-connect")).not.toBeInTheDocument();
  });

  it("connects a Bitbucket account with OAuth when Forge is configured", async () => {
    bitbucketOnboarding.providerConfigured = true;
    const user = userEvent.setup();
    const requestConnect = vi.fn();
    renderSetup(new Set(), { requestConnect });

    await chooseBitbucket(user);
    const link = screen.getByTestId("first-run-bitbucket-oauth");

    expect(link).toHaveAttribute("href", expect.stringContaining("/auth/bitbucket?intent=connect"));
    expect(requestConnect).not.toHaveBeenCalled();
    expect(
      screen.getByText("Connect your Bitbucket account. SuperPlane uses it to find repositories you can open."),
    ).toBeInTheDocument();
  });

  it("shows an existing installation without asking to install again when no repositories are available", async () => {
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
    bitbucketOnboarding.installedWorkspaces = [{ slug: "acme-team" }];
    const user = userEvent.setup();
    renderSetup(new Set());

    await chooseBitbucket(user);

    expect(screen.getByText("SuperPlane is installed on your Bitbucket workspace.")).toBeInTheDocument();
    expect(screen.getByText("Workspace: acme-team")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-bitbucket-install")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Check repositories" }));
    expect(bitbucketOnboarding.refetch).toHaveBeenCalled();
  });

  it("keeps checking when an installation is not yet detected", async () => {
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
    const user = userEvent.setup();
    const { rerender } = renderSetup(new Set());

    await chooseBitbucket(user);

    expect(screen.getByRole("status")).toHaveTextContent("SuperPlane checks automatically for installed workspaces.");
    expect(screen.getByRole("heading", { name: "Choose a Bitbucket workspace" })).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-stepper")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-bitbucket-waiting")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Check again" })).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-bitbucket-install")).toBeEnabled();

    bitbucketOnboarding.installedWorkspaces = [{ slug: "acme-team" }];
    bitbucketOnboarding.repositories = [{ fullName: "acme-team/api" }];
    rerender(bitbucketSetupElement());

    await user.click(await screen.findByRole("button", { name: "Use acme-team" }));

    expect(await screen.findByRole("option", { name: /acme-team\/api/ })).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-bitbucket-grant")).not.toBeInTheDocument();
  });

  it.each(["loading", "lookup failed", "connect", "grant", "choose"])(
    "keeps the %s controls separate from the Bitbucket animation",
    async (state) => {
      bitbucketOnboarding.providerConfigured = true;
      if (state === "loading") {
        bitbucketOnboarding.loaded = false;
        bitbucketOnboarding.isPending = true;
      }
      if (state === "lookup failed") {
        bitbucketOnboarding.loaded = false;
        bitbucketOnboarding.error = new Error("network error");
      }
      if (state === "grant" || state === "choose") {
        bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
      }
      if (state === "choose") {
        bitbucketOnboarding.repositories = [{ fullName: "acme-team/api" }];
      }
      const user = userEvent.setup();
      renderSetup(new Set());

      await chooseBitbucket(user);

      const artStage = screen.getByTestId("first-run-art-stage");
      expect(artStage).toContainElement(screen.getByTestId("first-run-art-pane"));
      expect(artStage).not.toContainElement(screen.getByTestId("first-run-content"));
      expect(screen.getByTestId("first-run-back")).toBeEnabled();
    },
  );

  it("restores the installation spinner after a refresh", async () => {
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
    writeBitbucketInstallWait("org-1", { startedAt: Date.now(), knownSlugs: [] });
    const user = userEvent.setup();
    renderSetup(new Set());

    await chooseBitbucket(user);

    expect(screen.getByTestId("first-run-bitbucket-waiting")).toHaveTextContent("Checking for the new workspace…");
  });

  it("keeps the remaining installation wait after a refresh", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-19T12:00:00.000Z"));
    try {
      bitbucketOnboarding.providerConfigured = true;
      bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
      const startedAt = Date.now() - 4 * 60_000;
      writeBitbucketInstallWait("org-1", { startedAt, knownSlugs: [] });
      const user = userEvent.setup({ delay: null, advanceTimers: vi.advanceTimersByTime });
      renderSetup(new Set());

      await chooseBitbucket(user);

      expect(screen.getByTestId("first-run-bitbucket-waiting")).toHaveTextContent("Checking for the new workspace…");
      expect(screen.queryByRole("button", { name: "Check again" })).not.toBeInTheDocument();
      expect(readBitbucketInstallWait("org-1")?.startedAt).toBe(startedAt);

      const remaining = startedAt + bitbucketInstallationAttemptMs - Date.now();
      expect(remaining).toBeGreaterThan(0);

      await act(async () => {
        await vi.advanceTimersByTimeAsync(remaining - 1);
      });

      expect(screen.getByTestId("first-run-bitbucket-waiting")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Check again" })).not.toBeInTheDocument();
      expect(readBitbucketInstallWait("org-1")?.startedAt).toBe(startedAt);

      await act(async () => {
        await vi.advanceTimersByTimeAsync(1);
      });

      expect(screen.queryByTestId("first-run-bitbucket-waiting")).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Check again" })).toBeInTheDocument();
      expect(readBitbucketInstallWait("org-1")?.startedAt).toBe(startedAt);
    } finally {
      vi.useRealTimers();
    }
  });

  it("shows Check again when a refreshed wait has already timed out", async () => {
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
    writeBitbucketInstallWait("org-1", {
      startedAt: Date.now() - bitbucketInstallationAttemptMs,
      knownSlugs: [],
    });
    const user = userEvent.setup();
    renderSetup(new Set());

    await chooseBitbucket(user);

    expect(screen.queryByTestId("first-run-bitbucket-waiting")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Check again" })).toBeInTheDocument();
    expect(readBitbucketInstallWait("org-1")).not.toBeNull();
  });

  it("starts a new wait when Check again is clicked after a timeout", async () => {
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
    writeBitbucketInstallWait("org-1", {
      startedAt: Date.now() - bitbucketInstallationAttemptMs,
      knownSlugs: ["old-team"],
    });
    const user = userEvent.setup();
    renderSetup(new Set());
    await chooseBitbucket(user);

    const beforeClick = Date.now();
    await user.click(screen.getByRole("button", { name: "Check again" }));

    expect(screen.getByTestId("first-run-bitbucket-waiting")).toHaveTextContent("Checking for the new workspace…");
    const saved = readBitbucketInstallWait("org-1");
    expect(saved?.startedAt).toBeGreaterThanOrEqual(beforeClick);
    expect(saved?.knownSlugs).toEqual([]);
  });

  it("stops the restored spinner when a new workspace appears", async () => {
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
    writeBitbucketInstallWait("org-1", { startedAt: Date.now(), knownSlugs: [] });
    const user = userEvent.setup();
    const { rerender } = renderSetup(new Set());
    await chooseBitbucket(user);
    expect(screen.getByTestId("first-run-bitbucket-waiting")).toBeInTheDocument();

    bitbucketOnboarding.installedWorkspaces = [{ slug: "acme-team" }];
    rerender(bitbucketSetupElement());

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Use acme-team" })).toBeInTheDocument();
      expect(screen.queryByText("Checking for the new workspace…")).not.toBeInTheDocument();
    });
    expect(readBitbucketInstallWait("org-1")).toBeNull();
  });

  it("does not restore a wait saved for another organization", async () => {
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
    writeBitbucketInstallWait("org-2", { startedAt: Date.now(), knownSlugs: [] });
    const user = userEvent.setup();
    renderSetup(new Set());

    await chooseBitbucket(user);

    expect(screen.queryByTestId("first-run-bitbucket-waiting")).not.toBeInTheDocument();
    expect(screen.getByText("SuperPlane checks automatically for installed workspaces.")).toBeInTheDocument();
  });

  it("offers a retry instead of the access token setup when the Bitbucket lookup fails", async () => {
    bitbucketOnboarding.loaded = false;
    bitbucketOnboarding.error = new Error("network error");
    const user = userEvent.setup();
    const requestConnect = vi.fn();
    renderSetup(new Set(), { requestConnect });

    await chooseBitbucket(user);

    expect(screen.queryByTestId("first-run-connect-bitbucket")).not.toBeInTheDocument();
    expect(screen.getByText("SuperPlane could not check the Bitbucket setup. Try again.")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-bitbucket-retry"));
    expect(bitbucketOnboarding.refetch).toHaveBeenCalled();
    expect(requestConnect).not.toHaveBeenCalled();
  });

  it("offers only Jira and Linear for a Bitbucket backlog", async () => {
    const user = userEvent.setup();
    const selectBitbucketRepository = vi.fn().mockResolvedValue(true);
    const requestConnect = vi.fn().mockResolvedValue(true);
    renderSetup(new Set<IntegrationId>(["bitbucket"]), {
      bitbucketIntegrationId: "bitbucket-1",
      selectBitbucketRepository,
      requestConnect,
    });

    await chooseBitbucket(user);
    await user.click(screen.getByRole("option", { name: /acme-team\/api/ }));
    await user.click(screen.getByTestId("first-run-continue-to-tickets"));

    await waitFor(() => expect(selectBitbucketRepository).toHaveBeenCalledWith("acme-team/api"));
    expect(resources.calls).toContainEqual({ integrationId: "bitbucket-1", type: "repository", enabled: true });
    expect(await screen.findByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /GitHub Issues/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect GitHub" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Jira/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Linear/ })).toBeInTheDocument();
    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();
    expect(requestConnect).not.toHaveBeenCalled();
  });

  it("returns from the Bitbucket repository screen to the host screen", async () => {
    const user = userEvent.setup();
    renderSetup(new Set<IntegrationId>(["bitbucket"]), { bitbucketIntegrationId: "bitbucket-1" });

    await chooseBitbucket(user);
    await user.click(screen.getByTestId("first-run-back"));

    expect(screen.getByTestId("first-run-host")).toBeInTheDocument();
  });
});
