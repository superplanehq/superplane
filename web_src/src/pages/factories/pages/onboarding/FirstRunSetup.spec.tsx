import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import {
  navigateSpy,
  pageModel,
  renderSetup,
  resetSetupFixtures,
  setupFixtures,
  setupState,
} from "./FirstRunSetup.testHelpers";

describe("FirstRunSetup", () => {
  beforeEach(() => {
    resetSetupFixtures();
  });

  it("finishes setup from the ticket screen when hosted credentials cover the agent", async () => {
    const user = userEvent.setup();
    const model = pageModel({ hostedAgentReady: true });

    renderSetup(model);

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze }));

    expect(model.saveIssues).toHaveBeenCalledWith("vcs");
    await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
    expect(screen.queryByTestId("first-run-agent")).not.toBeInTheDocument();
  });

  // Regression: the click that sets the issues choice and the call that
  // provisions the workspace happen in the same handler. `finish` used to
  // read the issues choice back off setup state captured before the click,
  // which was still empty, so it saved an empty issues source over the one
  // `saveIssues` had just stored and provisioning failed on the first click.
  // A repository with no issues took the same "vcs" (GitHub Issues) answer as
  // any other repository, so this reproduced on every repository, not only
  // ones without issues.
  it("passes the just-selected issues choice to finish instead of stale setup state", async () => {
    const user = userEvent.setup();
    const model = pageModel({ hostedAgentReady: true });

    renderSetup(model);

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze }));

    await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
    expect(model.finish).toHaveBeenCalledWith("vcs");
  });

  it("asks to connect GitHub when the workspace has not saved a connection", () => {
    setupFixtures.factory = { id: "factory-1", onboarding: {} };

    renderSetup(
      pageModel({
        openSection: "vcs",
        setup: { ...setupState(), vcsReady: true },
        selectedVcsConnectionId: "github-1",
        githubConnections: {
          name: "github",
          readyInstances: [
            {
              metadata: { id: "github-1", name: "github-acme", integrationName: "github" },
              status: { state: "ready", metadata: { owner: "acme" } },
            },
          ],
          allInstances: [
            {
              metadata: { id: "github-1", name: "github-acme", integrationName: "github" },
              status: { state: "ready", metadata: { owner: "acme" } },
            },
          ],
        },
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-connected")).not.toBeInTheDocument();
  });

  // A connection bound before picker data was kept has nothing to pick from,
  // so the screen offers a new connect instead of a dead connected state.
  it("asks to connect GitHub again when the saved connection kept no picker data", () => {
    renderSetup(
      pageModel({
        openSection: "vcs",
        setup: { ...setupState(), vcsReady: true },
        selectedVcsConnectionId: "github-1",
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-connected")).not.toBeInTheDocument();
  });

  // A resumed pending organization or workspace opens without a step in the
  // URL. Setup then always starts on the welcome screen, even when earlier
  // answers exist.
  it("starts on the welcome screen when the URL carries no step", () => {
    renderSetup(
      pageModel({
        openSection: "vcs",
        setup: { ...setupState(), vcsReady: true },
        selectedVcsConnectionId: "github-1",
      }),
      "/org-1/workspaces/PAY/setup",
    );

    expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect")).not.toBeInTheDocument();
  });

  // A refresh of the repository list must not show cached repositories from
  // an earlier connection while the fresh list loads.
  it("shows a placeholder on the repository screen while the list refreshes", () => {
    renderSetup(
      pageModel({ openSection: "repo", repositories: ["octo/stale-repo"], repositoriesLoading: true }),
      "/org-1/workspaces/PAY/setup?step=repo",
    );

    expect(screen.getByTestId("first-run-repositories-loading")).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /octo\/stale-repo/ })).not.toBeInTheDocument();
  });

  it("counts the ticket screen as the last step when the agent screen is skipped", () => {
    renderSetup(pageModel({ hostedAgentReady: true }));

    expect(screen.getByRole("navigation", { name: FIRST_RUN_COPY.chrome.stepLabel(4, 4) })).toBeInTheDocument();
  });

  it("shows setup progress on the ticket screen while it provisions the workspace", () => {
    renderSetup(pageModel({ hostedAgentReady: true, saving: true }));

    const finish = screen.getByTestId("first-run-analyze-tickets");
    expect(finish).toHaveTextContent(FIRST_RUN_COPY.finish.saving);
    expect(finish).toBeDisabled();
  });

  it("opens the agent screen when the agent needs a connected provider", async () => {
    const user = userEvent.setup();
    const model = pageModel({ hostedAgentReady: false });

    renderSetup(model);

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.continue }));

    expect(model.saveIssues).toHaveBeenCalledWith("vcs");
    expect(await screen.findByTestId("first-run-agent")).toBeInTheDocument();
    expect(model.finish).not.toHaveBeenCalled();
  });

  // Setup saved the ticket answer, then provisioning did not finish. The user
  // returns to the screen that carries the action, not to a screen with no
  // question left to answer.
  it("resumes on the ticket screen when hosted credentials cover the agent", () => {
    renderSetup(pageModel({ hostedAgentReady: true, openSection: "agent" }), "/org-1/workspaces/PAY/setup?step=agent");

    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-agent")).not.toBeInTheDocument();
  });

  it("resumes on the agent screen when the agent still needs a connected provider", () => {
    renderSetup(pageModel({ hostedAgentReady: false, openSection: "agent" }), "/org-1/workspaces/PAY/setup?step=agent");

    expect(screen.getByTestId("first-run-agent")).toBeInTheDocument();
  });

  it("shows Log out and the organization switch when another workspace exists", () => {
    setupFixtures.factories = [setupFixtures.factory, { id: "factory-2" }];

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-organization-switch")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-cancel")).not.toBeInTheDocument();
  });

  it("shows Log out and the organization switch when another organization exists", () => {
    setupFixtures.factories = [setupFixtures.factory];
    setupFixtures.accountOrganizations = [
      { id: "org-1", name: "Acme" },
      { id: "org-2", name: "Other Co" },
    ];

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-organization-switch")).toBeInTheDocument();
  });

  it("opens the current organization from the switch menu so the user can leave setup", async () => {
    setupFixtures.factories = [setupFixtures.factory, { id: "factory-2" }];
    const user = userEvent.setup();

    renderSetup(pageModel());

    await user.click(screen.getByTestId("first-run-organization-switch"));
    await user.click(screen.getByTestId("first-run-organization-option-org-1"));

    expect(navigateSpy).toHaveBeenCalledWith("/org-1");
  });

  it("goes back through every screen to the welcome screen", async () => {
    const user = userEvent.setup();
    const model = pageModel({
      setup: (() => {
        const setup = setupState();
        setup.selectRepo("acme/payments-service");
        return setup;
      })(),
    });

    renderSetup(model);

    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-choose")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
    // The welcome screen is the first screen, so it offers no Back.
    expect(screen.queryByTestId("first-run-back")).not.toBeInTheDocument();
  });

  it("keeps Log out and hides the organization switch with a single org and single workspace", () => {
    setupFixtures.factories = [setupFixtures.factory];
    setupFixtures.accountOrganizations = [{ id: "org-1", name: "Acme" }];

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-organization-switch")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-cancel")).not.toBeInTheDocument();
  });
});
