import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunChooseScreen } from "./FirstRunChooseScreen";

describe("FirstRunChooseScreen", () => {
  it("calls onGrantAccess when the grant access link is clicked", async () => {
    const user = userEvent.setup();
    const onGrantAccess = vi.fn();

    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onGrantAccess={onGrantAccess}
        onContinue={vi.fn()}
      />,
    );

    await user.click(screen.getByText(FIRST_RUN_COPY.choose.grantAccess));
    expect(onGrantAccess).toHaveBeenCalled();
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
        onGrantAccess={vi.fn()}
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
        onGrantAccess={vi.fn()}
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
        onGrantAccess={vi.fn()}
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
        onGrantAccess={vi.fn()}
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
        onGrantAccess={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-continue-to-tickets")).toHaveTextContent(FIRST_RUN_COPY.choose.saving);
    expect(screen.getByRole("option", { name: /octo\/repo/ })).toBeDisabled();
    expect(screen.getByText(FIRST_RUN_COPY.choose.grantAccess)).toBeDisabled();
  });

  it("shows the finished GitHub steps above the repository picker on the stepper card", () => {
    render(
      <FirstRunChooseScreen
        repositories={["puppies-inc/app"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onGrantAccess={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-github-stepper")).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.connect.connectGitHub)).toBeInTheDocument();
    expect(screen.queryByText("Grant access")).not.toBeInTheDocument();
    expect(screen.getAllByTestId("first-run-step-done")).toHaveLength(1);
    const repositoryList = screen.getByRole("listbox");
    expect(repositoryList).toHaveClass("max-h-48");
    expect(repositoryList).not.toHaveClass("max-h-56");
    expect(screen.getByRole("option", { name: /puppies-inc\/app/ })).toBeInTheDocument();
    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-content")).toHaveClass("py-16");
    expect(screen.getByTestId("first-run-content")).not.toHaveClass("py-24");
  });

  it("switches between linked GitHub accounts and can connect another account", async () => {
    const user = userEvent.setup();
    const onSelectGitHubIdentity = vi.fn();
    const onConnectAnotherGitHubAccount = vi.fn();

    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        githubLogin="forestileao"
        githubUserId="101"
        githubIdentities={[
          { userId: "101", login: "forestileao" },
          { userId: "202", login: "forestigamer" },
        ]}
        onSelectRepository={vi.fn()}
        onSelectGitHubIdentity={onSelectGitHubIdentity}
        onConnectAnotherGitHubAccount={onConnectAnotherGitHubAccount}
        onGrantAccess={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    const switchButton = screen.getByTestId("first-run-switch-github-account");
    expect(switchButton).toHaveTextContent(FIRST_RUN_COPY.choose.switchAccount);
    expect(switchButton).toHaveClass("text-muted-foreground");

    await user.click(switchButton);
    await user.click(screen.getByRole("menuitemradio", { name: "forestigamer" }));
    expect(onSelectGitHubIdentity).toHaveBeenCalledWith("202");

    await user.click(switchButton);
    await user.click(screen.getByRole("menuitem", { name: FIRST_RUN_COPY.choose.connectAnotherAccount }));
    expect(onConnectAnotherGitHubAccount).toHaveBeenCalled();
  });

  it("keeps repositories available while more repositories synchronize", () => {
    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        synchronizing
        onSelectRepository={vi.fn()}
        onGrantAccess={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByRole("option", { name: /octo\/repo/ })).toBeInTheDocument();
    const repositoryHeader = screen.getByTestId("first-run-step-repository-header");
    const status = within(repositoryHeader).getByTestId("first-run-repositories-synchronizing");
    expect(status).toHaveTextContent(FIRST_RUN_COPY.choose.synchronizing);
    expect(status).toHaveClass("sp-ai-thinking");
  });

  it("shows each repository without hiding earlier synchronization results", () => {
    const props = {
      selectedRepository: null,
      synchronizing: true,
      onSelectRepository: vi.fn(),
      onGrantAccess: vi.fn(),
      onContinue: vi.fn(),
    };
    const view = render(<FirstRunChooseScreen {...props} repositories={[]} />);

    expect(screen.queryByRole("option")).not.toBeInTheDocument();

    view.rerender(<FirstRunChooseScreen {...props} repositories={["acme/api"]} />);
    expect(screen.getByRole("option", { name: /acme\/api/ })).toBeInTheDocument();

    view.rerender(<FirstRunChooseScreen {...props} repositories={["acme/api", "acme/web"]} />);
    expect(screen.getByRole("option", { name: /acme\/api/ })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /acme\/web/ })).toBeInTheDocument();
  });

  it("shows every pending organization approval in the repository step", () => {
    render(
      <FirstRunChooseScreen
        repositories={[]}
        selectedRepository={null}
        pendingOrganizations={["acme", "example"]}
        onSelectRepository={vi.fn()}
        onGrantAccess={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByText("Waiting for approval for acme.")).toBeInTheDocument();
    expect(screen.getByText("Waiting for approval for example.")).toBeInTheDocument();
  });
});
