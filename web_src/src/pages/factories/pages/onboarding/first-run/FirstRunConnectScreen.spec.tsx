import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunConnectScreen } from "./FirstRunConnectScreen";

describe("FirstRunConnectScreen", () => {
  it("asks for GitHub identity before App access", async () => {
    const user = userEvent.setup();
    const connect = vi.fn();
    render(<FirstRunConnectScreen onConnectGitHub={connect} />);

    expect(screen.getByTestId("first-run-connect-github")).toHaveTextContent("Connect GitHub");
    expect(screen.getByText("Grant access")).toBeInTheDocument();
    expect(screen.getByText("Choose repository")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-connect-github"));
    expect(connect).toHaveBeenCalledTimes(1);
  });

  it("offers App access after GitHub identity is connected", () => {
    render(<FirstRunConnectScreen identityConnected githubLogin="octocat" onConnectGitHub={vi.fn()} />);

    expect(screen.getByTestId("first-run-github-signed-in-as")).toHaveTextContent("octocat");
    expect(screen.getByTestId("first-run-connect-github")).toHaveTextContent(FIRST_RUN_COPY.connect.grantAction);
  });

  it("shows every pending organization approval", () => {
    render(
      <FirstRunConnectScreen identityConnected pendingOrganizations={["acme", "example"]} onConnectGitHub={vi.fn()} />,
    );

    expect(screen.getByText("Waiting for approval for acme.")).toBeInTheDocument();
    expect(screen.getByText("Waiting for approval for example.")).toBeInTheDocument();
  });

  it("shows repository synchronization after an installation arrives", () => {
    render(<FirstRunConnectScreen identityConnected synchronizing onConnectGitHub={vi.fn()} />);

    expect(screen.getByRole("status")).toHaveTextContent(FIRST_RUN_COPY.connect.synchronizing);
  });
});
