import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunChooseScreen } from "./FirstRunChooseScreen";

describe("FirstRunChooseScreen", () => {
  it("calls onEditConnection when the edit connection link is clicked", async () => {
    const user = userEvent.setup();
    const onEditConnection = vi.fn();

    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onEditConnection={onEditConnection}
        onContinue={vi.fn()}
      />,
    );

    await user.click(screen.getByText(FIRST_RUN_COPY.choose.editConnection));
    expect(onEditConnection).toHaveBeenCalled();
  });

  it("calls onBack from the shell when Back is clicked", async () => {
    const user = userEvent.setup();
    const onBack = vi.fn();

    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository="octo/repo"
        chrome={{ stepIndex: 2, onBack }}
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("first-run-back"));
    expect(onBack).toHaveBeenCalled();
  });

  // A refresh after an edit of the GitHub connection must not show the old
  // cached repositories. The screen shows a placeholder until the list loads.
  it("shows a placeholder instead of stale repositories while loading", () => {
    render(
      <FirstRunChooseScreen
        repositories={["octo/stale-repo"]}
        selectedRepository={null}
        loading
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-repositories-loading")).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /octo\/stale-repo/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent(FIRST_RUN_COPY.choose.loading);
    expect(screen.getByTestId("first-run-repositories-spinner")).toBeInTheDocument();
  });

  it("disables continue until a repository is selected", () => {
    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    const continueButton = screen.getByTestId("first-run-continue-to-tickets");
    expect(continueButton).toBeDisabled();
    expect(continueButton).toHaveTextContent(FIRST_RUN_COPY.choose.continue);
  });

  it("enables continue and shows the ready label when a repository is selected", () => {
    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository="octo/repo"
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    const continueButton = screen.getByTestId("first-run-continue-to-tickets");
    expect(continueButton).toBeEnabled();
    expect(continueButton).toHaveTextContent(FIRST_RUN_COPY.choose.continueReady);
  });

  it("locks repository controls while the selection saves", () => {
    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository="octo/repo"
        saving
        chrome={{ stepIndex: 2, onBack: vi.fn() }}
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-continue-to-tickets")).toHaveTextContent(FIRST_RUN_COPY.choose.saving);
    expect(screen.getByRole("option", { name: /octo\/repo/ })).toBeDisabled();
    expect(screen.getByText(FIRST_RUN_COPY.choose.editConnection)).toBeDisabled();
  });

  it("shows the finished GitHub steps above the repository picker on the stepper card", () => {
    render(
      <FirstRunChooseScreen
        repositories={["puppies-inc/app"]}
        selectedRepository={null}
        stepper={{ organizationName: "puppies-inc" }}
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-stepper")).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.connect.stepConnected)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.connect.stepOrganizationDone("puppies-inc"))).toBeInTheDocument();
    expect(screen.getAllByTestId("first-run-step-done")).toHaveLength(2);
    expect(screen.getByRole("option", { name: /puppies-inc\/app/ })).toBeInTheDocument();
    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeInTheDocument();
  });
});
