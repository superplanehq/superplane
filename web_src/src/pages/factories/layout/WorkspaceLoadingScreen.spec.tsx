import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { WORKSPACE_LOADING_COPY, WORKSPACE_LOADING_TEST_ID } from "@/lib/workspaceLoadingCopy";

import { useWorkspaceLoading } from "@/hooks/useWorkspaceLoading";

import { WorkspaceLoadingScreen } from "./WorkspaceLoadingScreen";
import { WorkspaceLoadingProvider } from "./workspaceLoading";

function PendingReporter({ message, pending }: { message: string; pending: boolean }) {
  useWorkspaceLoading(message, pending);
  return <div>Ready child</div>;
}

describe("WorkspaceLoadingScreen", () => {
  it("shows the SuperPlane logo and the current message", () => {
    render(<WorkspaceLoadingScreen message={WORKSPACE_LOADING_COPY.board} />);

    const status = screen.getByRole("status", { name: WORKSPACE_LOADING_COPY.board });
    expect(status).toHaveAttribute("data-testid", WORKSPACE_LOADING_TEST_ID);
    expect(status).toHaveAttribute("aria-busy", "true");
    expect(status.querySelectorAll("svg path")).toHaveLength(4);
    expect(status.querySelector(".workspace-loading-pen")).not.toBeNull();
    expect(screen.getByText(WORKSPACE_LOADING_COPY.board)).toBeInTheDocument();
  });

  it("fades the overlay out when exiting", () => {
    render(<WorkspaceLoadingScreen message={WORKSPACE_LOADING_COPY.board} exiting />);

    const status = screen.getByTestId(WORKSPACE_LOADING_TEST_ID);
    expect(status).toHaveClass("workspace-loading-overlay--exit");
    expect(status).toHaveAttribute("aria-busy", "false");
  });
});

describe("WorkspaceLoadingProvider", () => {
  it("keeps one loading screen and updates the message", () => {
    const { rerender } = render(
      <WorkspaceLoadingProvider>
        <PendingReporter message={WORKSPACE_LOADING_COPY.workspace} pending />
      </WorkspaceLoadingProvider>,
    );

    expect(screen.getByTestId(WORKSPACE_LOADING_TEST_ID)).toHaveTextContent(WORKSPACE_LOADING_COPY.workspace);
    expect(screen.getAllByTestId(WORKSPACE_LOADING_TEST_ID)).toHaveLength(1);

    rerender(
      <WorkspaceLoadingProvider>
        <PendingReporter message={WORKSPACE_LOADING_COPY.board} pending />
      </WorkspaceLoadingProvider>,
    );

    expect(screen.getByTestId(WORKSPACE_LOADING_TEST_ID)).toHaveTextContent(WORKSPACE_LOADING_COPY.board);
    expect(screen.getAllByTestId(WORKSPACE_LOADING_TEST_ID)).toHaveLength(1);
  });

  it("keeps the same screen when one pending reporter replaces another", () => {
    const { rerender } = render(
      <WorkspaceLoadingProvider>
        <PendingReporter key="account" message={WORKSPACE_LOADING_COPY.account} pending />
      </WorkspaceLoadingProvider>,
    );

    const status = screen.getByTestId(WORKSPACE_LOADING_TEST_ID);
    expect(status).toHaveTextContent(WORKSPACE_LOADING_COPY.account);

    rerender(
      <WorkspaceLoadingProvider>
        <PendingReporter key="access" message={WORKSPACE_LOADING_COPY.access} pending />
      </WorkspaceLoadingProvider>,
    );

    const nextStatus = screen.getByTestId(WORKSPACE_LOADING_TEST_ID);
    expect(nextStatus).toBe(status);
    expect(nextStatus).toHaveTextContent(WORKSPACE_LOADING_COPY.access);
    expect(screen.getAllByTestId(WORKSPACE_LOADING_TEST_ID)).toHaveLength(1);
  });

  it("fades the screen out when nothing is pending", () => {
    const { rerender } = render(
      <WorkspaceLoadingProvider>
        <PendingReporter message={WORKSPACE_LOADING_COPY.board} pending />
      </WorkspaceLoadingProvider>,
    );

    rerender(
      <WorkspaceLoadingProvider>
        <PendingReporter message={WORKSPACE_LOADING_COPY.board} pending={false} />
      </WorkspaceLoadingProvider>,
    );

    expect(screen.getByTestId(WORKSPACE_LOADING_TEST_ID)).toHaveClass("workspace-loading-overlay--exit");
    expect(screen.getByText("Ready child")).toBeInTheDocument();
  });
});
