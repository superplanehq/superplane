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

  it("offers a private GitHub App when hosted install is the default", async () => {
    const user = userEvent.setup();
    const onCreatePrivateApp = vi.fn();

    render(
      <FirstRunConnectScreen
        githubConnected={false}
        showPrivateApp
        onConnectGitHub={vi.fn()}
        onCreatePrivateApp={onCreatePrivateApp}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-create-private-github-app")).toHaveTextContent(
      FIRST_RUN_COPY.connect.createPrivateApp,
    );
    await user.click(screen.getByTestId("first-run-create-private-github-app"));
    expect(onCreatePrivateApp).toHaveBeenCalled();
  });

  it("continues to the repository step after GitHub is connected", async () => {
    const user = userEvent.setup();
    const onContinue = vi.fn();

    render(
      <FirstRunConnectScreen
        githubConnected
        showPrivateApp
        onConnectGitHub={vi.fn()}
        onCreatePrivateApp={vi.fn()}
        onContinue={onContinue}
      />,
    );

    expect(screen.getByTestId("first-run-github-connected")).toHaveTextContent(FIRST_RUN_COPY.connect.connected);
    expect(screen.queryByTestId("first-run-create-private-github-app")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-github-continue"));
    expect(onContinue).toHaveBeenCalled();
  });
});
