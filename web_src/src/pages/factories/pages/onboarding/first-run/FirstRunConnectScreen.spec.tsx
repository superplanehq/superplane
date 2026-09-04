import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunConnectScreen } from "./FirstRunConnectScreen";

describe("FirstRunConnectScreen", () => {
  it("asks only for GitHub and keeps tickets off this screen", async () => {
    const user = userEvent.setup();
    const onConnectGitHub = vi.fn();

    render(<FirstRunConnectScreen githubConnected={false} onConnectGitHub={onConnectGitHub} onContinue={vi.fn()} />);

    expect(screen.getByTestId("first-run-connect-github")).toHaveTextContent(FIRST_RUN_COPY.connect.connectGitHub);
    expect(screen.getByText(FIRST_RUN_COPY.connect.trust)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.tickets.trust)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /GitHub Issues/ })).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-create-private-github-app")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-connect-github"));
    expect(onConnectGitHub).toHaveBeenCalled();
  });

  it("never shows the private GitHub App option, connected or not", () => {
    const { rerender } = render(
      <FirstRunConnectScreen githubConnected={false} onConnectGitHub={vi.fn()} onContinue={vi.fn()} />,
    );
    expect(screen.queryByTestId("first-run-create-private-github-app")).not.toBeInTheDocument();

    rerender(<FirstRunConnectScreen githubConnected onConnectGitHub={vi.fn()} onContinue={vi.fn()} />);
    expect(screen.queryByTestId("first-run-create-private-github-app")).not.toBeInTheDocument();
  });

  it("explains a pending GitHub install request without treating it as an error", async () => {
    const user = userEvent.setup();
    const onConnectGitHub = vi.fn();

    render(
      <FirstRunConnectScreen
        githubConnected={false}
        installRequested
        onConnectGitHub={onConnectGitHub}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-install-requested")).toHaveTextContent(
      FIRST_RUN_COPY.connect.installRequested,
    );
    expect(screen.queryByText(FIRST_RUN_COPY.connect.installRequestedBody())).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-install-org")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.connect.connectError)).not.toBeInTheDocument();
    expect(document.querySelector(".text-destructive")).not.toBeInTheDocument();

    await user.hover(screen.getByTestId("first-run-github-install-help"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent(FIRST_RUN_COPY.connect.installRequestedBody());
    expect(screen.getByRole("tooltip")).toHaveTextContent(FIRST_RUN_COPY.connect.installRequestedNext);

    await user.click(screen.getByTestId("first-run-connect-github"));
    expect(onConnectGitHub).toHaveBeenCalled();
  });

  it("hides a connect error while the install request is waiting", () => {
    render(
      <FirstRunConnectScreen
        githubConnected={false}
        installRequested
        connectError={FIRST_RUN_COPY.connect.connectError}
        onConnectGitHub={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-install-requested")).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.connect.connectError)).not.toBeInTheDocument();
    expect(document.querySelector(".text-destructive")).not.toBeInTheDocument();
  });

  it("names the GitHub organization that is waiting for approval", () => {
    render(
      <FirstRunConnectScreen
        githubConnected={false}
        installRequested
        githubOrganization="acme"
        onConnectGitHub={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-install-org")).toHaveTextContent("acme");
    expect(screen.queryByText(FIRST_RUN_COPY.connect.installRequestedBody("acme"))).not.toBeInTheDocument();
  });

  it("names the GitHub organization in the waiting tooltip", async () => {
    const user = userEvent.setup();

    render(
      <FirstRunConnectScreen
        githubConnected={false}
        installRequested
        githubOrganization="acme"
        onConnectGitHub={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    await user.hover(screen.getByTestId("first-run-github-install-help"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent(FIRST_RUN_COPY.connect.installRequestedBody("acme"));
  });

  it("continues to the repository step after GitHub is connected", async () => {
    const user = userEvent.setup();
    const onContinue = vi.fn();

    render(<FirstRunConnectScreen githubConnected onConnectGitHub={vi.fn()} onContinue={onContinue} />);

    expect(screen.getByTestId("first-run-github-connected")).toHaveTextContent(FIRST_RUN_COPY.connect.connected);
    expect(screen.queryByTestId("first-run-create-private-github-app")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-github-continue"));
    expect(onContinue).toHaveBeenCalled();
  });
});
