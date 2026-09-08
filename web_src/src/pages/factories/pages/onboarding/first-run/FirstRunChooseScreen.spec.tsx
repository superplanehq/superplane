import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunChooseScreen } from "./FirstRunChooseScreen";

describe("FirstRunChooseScreen", () => {
  it("shows an access hint next to the edit connection link", () => {
    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-choose-access-hint")).toHaveTextContent(FIRST_RUN_COPY.choose.accessHint);
  });

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

  it("reveals why a repository might not show up when expanded", async () => {
    const user = userEvent.setup();

    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    const disclosure = screen.getByTestId("first-run-choose-why-missing");
    expect(screen.getByText(FIRST_RUN_COPY.choose.missingTitle)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.choose.missingReasons[0])).not.toBeVisible();

    await user.click(screen.getByText(FIRST_RUN_COPY.choose.missingTitle));

    expect(disclosure).toHaveAttribute("open");
    for (const reason of FIRST_RUN_COPY.choose.missingReasons) {
      expect(screen.getByText(reason)).toBeVisible();
    }
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
});
