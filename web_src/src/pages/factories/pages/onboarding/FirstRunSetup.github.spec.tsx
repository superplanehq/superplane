import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import {
  bindMutate,
  pageModel,
  renderSetup,
  resetSetupFixtures,
  setupFixtures,
  setupState,
  type OnboardingPageModel,
} from "./FirstRunSetup.testHelpers";

describe("FirstRunSetup GitHub picker", () => {
  beforeEach(() => {
    resetSetupFixtures();
  });

  /** The workspace's bound connection, with the picker data a bind keeps. */
  function boundConnectionModel(selectVcsConnection: OnboardingPageModel["selectVcsConnection"]) {
    const boundInstance = {
      metadata: { id: "github-1", name: "github-acme", integrationName: "github" },
      status: {
        state: "ready",
        metadata: {
          owner: "acme",
          startedByUserID: "user-1",
          startedByGitHubLogin: "forestileao",
          state: "csrf",
          githubApp: { slug: "superplane" },
          pendingInstallations: [
            { id: "11", accountLogin: "acme" },
            { id: "22", accountLogin: "octo" },
          ],
        },
      },
    };
    return pageModel({
      openSection: "vcs",
      setup: { ...setupState(), vcsReady: true },
      selectedVcsConnectionId: "github-1",
      selectVcsConnection,
      githubConnections: {
        name: "github",
        readyInstances: [boundInstance],
        allInstances: [boundInstance],
      },
    });
  }

  function bindablePageModel(selectVcsConnection: OnboardingPageModel["selectVcsConnection"]) {
    const pendingInstance = {
      metadata: { id: "int-new", integrationName: "github" },
      status: {
        state: "pending",
        metadata: {
          startedByUserID: "user-1",
          state: "csrf",
          githubApp: { slug: "superplane" },
          pendingInstallations: [{ id: "11", accountLogin: "acme" }],
        },
      },
    };
    // The static test model shows the post-bind refetch already applied: the
    // bound connection reports ready.
    const readyInstance = {
      metadata: { id: "int-new", name: "github-acme", integrationName: "github" },
      status: { state: "ready", metadata: { owner: "acme" } },
    };
    return pageModel({
      openSection: "vcs",
      selectVcsConnection,
      githubConnections: {
        name: "github",
        readyInstances: [readyInstance],
        allInstances: [pendingInstance],
      },
    });
  }

  it("reopens the account picker for the workspace's bound connection", () => {
    renderSetup(boundConnectionModel(vi.fn().mockResolvedValue(true)), "/org-1/workspaces/PAY/setup?step=vcs");

    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("octo") })).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-signed-in-as")).toHaveTextContent(
      FIRST_RUN_COPY.connect.signedInAs("forestileao"),
    );
    expect(screen.queryByTestId("first-run-github-connected")).not.toBeInTheDocument();
  });

  // Get started always opens the Connect GitHub page, even when picker data
  // exists. The user picks the GitHub account on every forward pass, because
  // many people stay signed in to two GitHub accounts.
  it("opens the Connect GitHub page from Get started even when picker data exists", async () => {
    const user = userEvent.setup();

    renderSetup(boundConnectionModel(vi.fn().mockResolvedValue(true)), "/org-1/workspaces/PAY/setup");

    expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-get-started"));

    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-account-picker")).not.toBeInTheDocument();
  });

  // Back walks the exact pages in reverse order: account picker, Connect
  // GitHub page, welcome.
  it("walks back from the picker to the Connect GitHub page and then to welcome", async () => {
    const user = userEvent.setup();

    renderSetup(boundConnectionModel(vi.fn().mockResolvedValue(true)), "/org-1/workspaces/PAY/setup?step=vcs");

    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));

    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-account-picker")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));

    expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
  });

  it("reopens the account picker when the user goes back from the repository screen", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(true);
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(boundConnectionModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("octo") }));
    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
  });

  // A GitHub round trip reloads the page, so the picker data arrives after
  // the first render. The screen must not flash the connect button first.
  it("shows a placeholder on the connect screen while the connection list loads", () => {
    renderSetup(
      pageModel({ openSection: "vcs", githubConnectionsLoading: true }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.getByTestId("first-run-connect-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
  });

  it("moves the bound connection to another account through the picker", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(true);
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(boundConnectionModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("octo") }));

    expect(bindMutate).toHaveBeenCalledWith(
      { state: "csrf", installationId: "22" },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
    await waitFor(() => expect(selectVcsConnection).toHaveBeenCalledWith("github-1"));
    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
  });

  it("saves the bound GitHub connection for a workspace of an existing organization", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(true);
    setupFixtures.factory = { id: "factory-1", onboarding: {} };
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(bindablePageModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") }));

    await waitFor(() => expect(selectVcsConnection).toHaveBeenCalledWith("int-new"));
    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
  });

  it("saves the bound GitHub connection for the initial organization", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(true);
    setupFixtures.factory = { id: "factory-1", onboarding: { initial: true } };
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(bindablePageModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") }));

    await waitFor(() => expect(selectVcsConnection).toHaveBeenCalledWith("int-new"));
    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
  });

  // Regression: the repository screen opened before the bound connection was
  // saved, so a fast repository pick stored the repository on the prior
  // connection. The screen must stay on connect when the save fails.
  it("keeps the connect screen when the bound connection does not save", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(false);
    setupFixtures.factory = { id: "factory-1", onboarding: {} };
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(bindablePageModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") }));

    await waitFor(() => expect(selectVcsConnection).toHaveBeenCalledWith("int-new"));
    expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-choose")).not.toBeInTheDocument();
  });

  it("shows the GitHub account picker on the connect screen", () => {
    renderSetup(
      pageModel({
        openSection: "vcs",
        githubConnections: {
          name: "github",
          readyInstances: [],
          allInstances: [
            {
              metadata: { id: "int-1", integrationName: "github" },
              status: {
                state: "pending",
                metadata: {
                  startedByUserID: "user-1",
                  state: "csrf",
                  githubApp: { slug: "superplane" },
                  pendingInstallations: [
                    { id: "11", accountLogin: "acme" },
                    { id: "22", accountLogin: "octo" },
                  ],
                },
              },
            },
          ],
        },
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.getByRole("heading", { name: FIRST_RUN_COPY.connect.selectAccount })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") })).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
  });

  it("does not show another member's GitHub account picker", () => {
    renderSetup(
      pageModel({
        openSection: "vcs",
        githubConnections: {
          name: "github",
          readyInstances: [],
          allInstances: [
            {
              metadata: { id: "int-1", integrationName: "github" },
              status: {
                state: "pending",
                metadata: {
                  startedByUserID: "some-other-user",
                  state: "csrf",
                  githubApp: { slug: "superplane" },
                  pendingInstallations: [
                    { id: "11", accountLogin: "acme" },
                    { id: "22", accountLogin: "octo" },
                  ],
                },
              },
            },
          ],
        },
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.queryByTestId("first-run-github-account-picker")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
  });

  it("opens Connect when GitHub returned an install request without a step", () => {
    renderSetup(pageModel({ openSection: "vcs" }), "/org-1/workspaces/PAY/setup?githubSetup=request");

    expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-install-requested")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-welcome")).not.toBeInTheDocument();
  });

  it("shows a waiting chip when GitHub returned an install request", () => {
    renderSetup(pageModel({ openSection: "vcs" }), "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=request");

    expect(screen.getByTestId("first-run-github-install-requested")).toHaveTextContent(
      FIRST_RUN_COPY.connect.installRequested,
    );
    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
  });

  it("shows a waiting account row when the pending connection has a request and no ready account", () => {
    const pendingInstance = {
      metadata: { id: "int-1", integrationName: "github" },
      status: {
        state: "pending",
        metadata: {
          startedByUserID: "user-1",
          startedByGitHubLogin: "ada",
          installRequested: true,
          installRequestedAccount: "acme",
          state: "csrf",
          githubApp: { slug: "superplane" },
        },
      },
    };

    renderSetup(
      pageModel({
        openSection: "vcs",
        githubConnections: { name: "github", allInstances: [pendingInstance], readyInstances: [] },
      }),
      "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=request&githubOrg=acme",
    );

    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-install-org")).toHaveTextContent("acme");
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
  });

  it("names the GitHub organization from the return query", () => {
    renderSetup(
      pageModel({ openSection: "vcs" }),
      "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=request&githubOrg=acme",
    );

    expect(screen.getByTestId("first-run-github-install-org")).toHaveTextContent("acme");
    expect(screen.queryByText(FIRST_RUN_COPY.connect.installRequestedBody("acme"))).not.toBeInTheDocument();
  });
});
