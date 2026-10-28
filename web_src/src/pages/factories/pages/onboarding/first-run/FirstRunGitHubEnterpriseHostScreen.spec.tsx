import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGitHubEnterpriseHostScreen } from "./FirstRunGitHubEnterpriseHostScreen";

describe("FirstRunGitHubEnterpriseHostScreen", () => {
  it("asks for the GitHub Enterprise Cloud hostname", async () => {
    const user = userEvent.setup();
    const onContinueHost = vi.fn();
    render(<FirstRunGitHubEnterpriseHostScreen onContinueHost={onContinueHost} onContinueOrganization={vi.fn()} />);

    expect(screen.getByTestId("first-run-github-enterprise-host")).toHaveTextContent(
      FIRST_RUN_COPY.githubEnterprise.headline,
    );
    expect(screen.getByTestId("first-run-github-enterprise-continue")).toBeDisabled();

    await user.type(screen.getByTestId("first-run-github-enterprise-hostname"), "acme.ghe.com");
    await user.click(screen.getByTestId("first-run-github-enterprise-continue"));

    expect(onContinueHost).toHaveBeenCalledWith("acme.ghe.com");
    expect(screen.getByRole("link", { name: FIRST_RUN_COPY.githubEnterprise.ipAllowListLink })).toHaveAttribute(
      "href",
      FIRST_RUN_COPY.githubEnterprise.ipAllowListHref,
    );
  });

  it("asks for the GitHub organization after the hostname is accepted", async () => {
    const user = userEvent.setup();
    const onContinueOrganization = vi.fn();
    render(
      <FirstRunGitHubEnterpriseHostScreen
        phase="organization"
        onContinueHost={vi.fn()}
        onContinueOrganization={onContinueOrganization}
      />,
    );

    expect(screen.queryByTestId("first-run-github-enterprise-hostname")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-enterprise-host")).toHaveTextContent(
      FIRST_RUN_COPY.githubEnterprise.organizationHeadline,
    );
    expect(screen.getByTestId("first-run-github-enterprise-create-app")).toBeDisabled();

    await user.type(screen.getByTestId("first-run-github-enterprise-organization"), "acme");
    await user.click(screen.getByTestId("first-run-github-enterprise-create-app"));

    expect(onContinueOrganization).toHaveBeenCalledWith("acme");
  });

  it("keeps a pending App on the finish-setup screen", async () => {
    const user = userEvent.setup();
    const onResumeSetup = vi.fn();
    render(
      <FirstRunGitHubEnterpriseHostScreen
        phase="app"
        onContinueHost={vi.fn()}
        onContinueOrganization={vi.fn()}
        onResumeSetup={onResumeSetup}
      />,
    );

    expect(screen.queryByTestId("first-run-github-enterprise-hostname")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-enterprise-organization")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-enterprise-host")).toHaveTextContent(
      FIRST_RUN_COPY.githubEnterprise.finishHeadline,
    );
    await user.click(screen.getByTestId("first-run-github-enterprise-finish"));
    expect(onResumeSetup).toHaveBeenCalled();
  });

  it("shows a connection error so the user can retry", () => {
    render(
      <FirstRunGitHubEnterpriseHostScreen
        connectError={FIRST_RUN_COPY.githubEnterprise.connectError}
        onContinueHost={vi.fn()}
        onContinueOrganization={vi.fn()}
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent(FIRST_RUN_COPY.githubEnterprise.connectError);
  });
});
