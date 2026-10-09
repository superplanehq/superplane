import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FirstRunConnectScreen } from "./FirstRunConnectScreen";
import { FIRST_RUN_COPY } from "./firstRunCopy";

describe("FirstRunConnectScreen", () => {
  it("asks for GitHub identity before repository selection", async () => {
    const user = userEvent.setup();
    const connect = vi.fn();
    render(<FirstRunConnectScreen onConnectGitHub={connect} />);

    expect(screen.getByTestId("first-run-connect-github")).toHaveTextContent("Connect GitHub");
    expect(screen.getByText("Choose repository")).toBeInTheDocument();
    expect(screen.queryByText("Grant access")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-connect-github"));
    expect(connect).toHaveBeenCalledTimes(1);
  });

  it("asks to create the GitHub App when the installation has none", async () => {
    const user = userEvent.setup();
    const connect = vi.fn();
    render(<FirstRunConnectScreen createApp onConnectGitHub={connect} />);

    expect(screen.getByTestId("first-run-connect-github")).toHaveTextContent("Create GitHub App");
    expect(screen.getByText(FIRST_RUN_COPY.connect.createAppBody)).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-connect-github"));
    expect(connect).toHaveBeenCalledTimes(1);
  });

});
