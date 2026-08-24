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

    await user.click(screen.getByTestId("first-run-connect-github"));
    expect(onConnectGitHub).toHaveBeenCalled();
  });

  it("continues to the repository step after GitHub is connected", async () => {
    const user = userEvent.setup();
    const onContinue = vi.fn();

    render(<FirstRunConnectScreen githubConnected onConnectGitHub={vi.fn()} onContinue={onContinue} />);

    expect(screen.getByTestId("first-run-github-connected")).toHaveTextContent(FIRST_RUN_COPY.connect.connected);
    await user.click(screen.getByTestId("first-run-github-continue"));
    expect(onContinue).toHaveBeenCalled();
  });
});
