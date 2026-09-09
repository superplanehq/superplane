import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunConnectScreen } from "./FirstRunConnectScreen";

describe("FirstRunConnectScreen", () => {
  it("asks only for GitHub and keeps tickets off this screen", async () => {
    const user = userEvent.setup();
    const onConnectGitHub = vi.fn();

    render(<FirstRunConnectScreen onConnectGitHub={onConnectGitHub} />);

    expect(screen.getByTestId("first-run-connect-github")).toHaveTextContent(FIRST_RUN_COPY.connect.connectGitHub);
    expect(screen.getByText(FIRST_RUN_COPY.connect.trust)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.tickets.trust)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /GitHub Issues/ })).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-create-private-github-app")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-connect-github"));
    expect(onConnectGitHub).toHaveBeenCalled();
  });

  it("never shows the private GitHub App option", () => {
    render(<FirstRunConnectScreen onConnectGitHub={vi.fn()} />);
    expect(screen.queryByTestId("first-run-create-private-github-app")).not.toBeInTheDocument();
  });

  it("explains a pending GitHub install request without treating it as an error", async () => {
    const user = userEvent.setup();
    const onConnectGitHub = vi.fn();

    render(<FirstRunConnectScreen installRequested onConnectGitHub={onConnectGitHub} />);

    expect(screen.getByTestId("first-run-github-install-requested")).toHaveTextContent(
      FIRST_RUN_COPY.connect.installRequested,
    );
    expect(screen.queryByText(FIRST_RUN_COPY.connect.installRequestedBody())).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-install-org")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-install-help")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.connect.installRequestedNext)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.connect.connectError)).not.toBeInTheDocument();
    expect(document.querySelector(".text-destructive")).not.toBeInTheDocument();

    await user.hover(screen.getByTestId("first-run-github-install-requested"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent(FIRST_RUN_COPY.connect.installRequestedBody());
    expect(screen.getByRole("tooltip")).toHaveTextContent(FIRST_RUN_COPY.connect.installRequestedNext);

    await user.click(screen.getByTestId("first-run-connect-github"));
    expect(onConnectGitHub).toHaveBeenCalled();
  });

  it("hides a connect error while the install request is waiting", () => {
    render(
      <FirstRunConnectScreen
        installRequested
        connectError={FIRST_RUN_COPY.connect.connectError}
        onConnectGitHub={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-install-requested")).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.connect.connectError)).not.toBeInTheDocument();
    expect(document.querySelector(".text-destructive")).not.toBeInTheDocument();
  });

  it("names the GitHub organization that is waiting for approval", () => {
    render(<FirstRunConnectScreen installRequested githubOrganization="acme" onConnectGitHub={vi.fn()} />);

    expect(screen.getByTestId("first-run-github-install-org")).toHaveTextContent("acme");
    expect(screen.queryByText(FIRST_RUN_COPY.connect.installRequestedBody("acme"))).not.toBeInTheDocument();
  });

  it("names the GitHub organization in the waiting tooltip", async () => {
    const user = userEvent.setup();

    render(<FirstRunConnectScreen installRequested githubOrganization="acme" onConnectGitHub={vi.fn()} />);

    await user.hover(screen.getByTestId("first-run-github-install-requested"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent(FIRST_RUN_COPY.connect.installRequestedBody("acme"));
  });

  it("names the GitHub login that authorized the connect when the picker shows", () => {
    render(
      <FirstRunConnectScreen
        pendingInstallations={[{ id: "11", accountLogin: "octo" }]}
        githubState="csrf"
        githubAppSlug="superplane"
        githubLogin="forestileao"
        onConnectGitHub={vi.fn()}
        onUseInstallation={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-signed-in-as")).toHaveTextContent(
      FIRST_RUN_COPY.connect.signedInAs("forestileao"),
    );
  });

  it("hides the signed-in line when the picker has no GitHub login", () => {
    render(
      <FirstRunConnectScreen
        pendingInstallations={[{ id: "11", accountLogin: "octo" }]}
        githubState="csrf"
        githubAppSlug="superplane"
        onConnectGitHub={vi.fn()}
        onUseInstallation={vi.fn()}
      />,
    );

    expect(screen.queryByTestId("first-run-github-signed-in-as")).not.toBeInTheDocument();
  });

  it("asks which GitHub account to use when one install is pending", () => {
    render(
      <FirstRunConnectScreen
        pendingInstallations={[{ id: "11", accountLogin: "octo" }]}
        githubState="csrf"
        githubAppSlug="superplane"
        onConnectGitHub={vi.fn()}
        onUseInstallation={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-account-picker")).toHaveTextContent(
      FIRST_RUN_COPY.connect.selectAccount,
    );
    expect(screen.getByTestId("first-run-github-use-octo")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.connect.missingAccount)).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-install-other")).toHaveTextContent(FIRST_RUN_COPY.connect.installThere);
  });

  it("asks which GitHub account to use when two installs are pending", async () => {
    const user = userEvent.setup();
    const onUseInstallation = vi.fn();

    render(
      <FirstRunConnectScreen
        pendingInstallations={[
          { id: "11", accountLogin: "acme" },
          { id: "22", accountLogin: "octo" },
        ]}
        githubState="csrf"
        githubAppSlug="superplane"
        onConnectGitHub={vi.fn()}
        onUseInstallation={onUseInstallation}
      />,
    );

    expect(screen.getByTestId("first-run-github-account-picker")).toHaveTextContent(
      FIRST_RUN_COPY.connect.selectAccount,
    );
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-install-other")).toHaveAttribute(
      "href",
      "https://github.com/apps/superplane/installations/new?state=csrf",
    );

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") }));
    expect(onUseInstallation).toHaveBeenCalledWith({ id: "11", accountLogin: "acme" });
  });

  it("hides the waiting chip when the picker offers the requested organization", () => {
    render(
      <FirstRunConnectScreen
        installRequested
        githubOrganization="Acme"
        pendingInstallations={[{ id: "11", accountLogin: "acme" }]}
        githubState="csrf"
        githubAppSlug="superplane"
        onConnectGitHub={vi.fn()}
        onUseInstallation={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-install-requested")).not.toBeInTheDocument();
  });

  it("keeps the waiting chip when the picker lacks the requested organization", () => {
    render(
      <FirstRunConnectScreen
        installRequested
        githubOrganization="acme"
        pendingInstallations={[{ id: "22", accountLogin: "octo" }]}
        githubState="csrf"
        githubAppSlug="superplane"
        onConnectGitHub={vi.fn()}
        onUseInstallation={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-install-requested")).toBeInTheDocument();
  });

  it("disables the picker while one account is binding", () => {
    render(
      <FirstRunConnectScreen
        pendingInstallations={[
          { id: "11", accountLogin: "acme" },
          { id: "22", accountLogin: "octo" },
        ]}
        githubState="csrf"
        githubAppSlug="superplane"
        bindingInstallationId="11"
        onConnectGitHub={vi.fn()}
        onUseInstallation={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-use-acme")).toBeDisabled();
    expect(screen.getByTestId("first-run-github-use-octo")).toBeDisabled();
  });

  // A GitHub round trip reloads the page, so the picker data arrives after
  // the first render. The placeholder keeps the screen from flashing the
  // connect button before the picker.
  it("shows a placeholder instead of the connect button while loading", () => {
    render(<FirstRunConnectScreen loading onConnectGitHub={vi.fn()} />);

    expect(screen.getByTestId("first-run-connect-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-account-picker")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent(FIRST_RUN_COPY.connect.loadingAccounts);
  });

  it("shows progress and prevents another connect while GitHub opens", () => {
    render(<FirstRunConnectScreen connecting onConnectGitHub={vi.fn()} />);

    const connect = screen.getByTestId("first-run-connect-github");
    expect(connect).toHaveTextContent(FIRST_RUN_COPY.connect.openingGitHub);
    expect(connect).toBeDisabled();
  });

  it("never shows a connected state; the picker or the connect button always shows", () => {
    render(<FirstRunConnectScreen onConnectGitHub={vi.fn()} />);

    expect(screen.queryByTestId("first-run-github-connected")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-continue")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
  });
});
