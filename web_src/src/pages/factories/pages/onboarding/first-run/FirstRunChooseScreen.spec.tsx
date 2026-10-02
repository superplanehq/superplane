import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunChooseScreen } from "./FirstRunChooseScreen";

const copy = FIRST_RUN_COPY.choose;

function chooseOrganization(user: ReturnType<typeof userEvent.setup>, organization: string) {
  return user.click(screen.getByRole("button", { name: copy.useOrganization(organization) }));
}

describe("FirstRunChooseScreen", () => {
  describe("organization step", () => {
    it("offers one button for each repository owner", () => {
      render(
        <FirstRunChooseScreen
          repositories={["forestileao/test", "superplanehq/superplane", "forestileao/instabot"]}
          selectedRepository={null}
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByRole("heading", { name: copy.organizationHeadline })).toBeInTheDocument();
      const picker = screen.getByTestId("first-run-github-organization-picker");
      expect(
        within(picker)
          .getAllByRole("button", { name: /^Use / })
          .map((button) => button.textContent),
      ).toEqual(["Use forestileao", "Use superplanehq"]);
      expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
      expect(screen.queryByTestId("first-run-continue-to-tickets")).not.toBeInTheDocument();
    });

    it("shows the step when only one organization owns the repositories", () => {
      render(
        <FirstRunChooseScreen
          repositories={["octo/repo"]}
          selectedRepository={null}
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByRole("button", { name: copy.useOrganization("octo") })).toBeInTheDocument();
      expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });

    it("offers GitHub installation for a missing organization", async () => {
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

      expect(screen.getByText(copy.missingOrganization)).toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: copy.installAction }));
      expect(onGrantAccess).toHaveBeenCalled();
    });

    it("offers GitHub installation as the main action when no repositories are available", async () => {
      const user = userEvent.setup();
      const onGrantAccess = vi.fn();

      render(
        <FirstRunChooseScreen
          repositories={[]}
          selectedRepository={null}
          onSelectRepository={vi.fn()}
          onGrantAccess={onGrantAccess}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByRole("heading", { name: copy.organizationHeadline })).toBeInTheDocument();
      expect(screen.getByText(copy.emptyTitle)).toBeInTheDocument();
      expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
      expect(screen.queryByTestId("first-run-continue-to-tickets")).not.toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: copy.installAction }));
      expect(onGrantAccess).toHaveBeenCalled();
    });

    it("shows every pending organization approval", () => {
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
      expect(screen.getByText(/Ask an admin of acme to approve the SuperPlane GitHub App\./)).toBeVisible();
      expect(screen.getByTestId("first-run-grant-access")).toHaveTextContent(copy.installAnotherAction);
    });

    it("keeps pending approvals visible next to the available organizations", () => {
      render(
        <FirstRunChooseScreen
          repositories={["octo/repo"]}
          selectedRepository={null}
          pendingOrganizations={["acme"]}
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByText("Waiting for approval for acme.")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: copy.useOrganization("octo") })).toBeInTheDocument();
    });

    it("shows generic approval copy when GitHub does not name the organization", () => {
      render(
        <FirstRunChooseScreen
          repositories={[]}
          selectedRepository={null}
          pendingOrganizations={[""]}
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByText("Waiting for GitHub approval.")).toBeInTheDocument();
      expect(screen.getByText(/Ask a GitHub organization admin to approve the SuperPlane GitHub App\./)).toBeVisible();
    });

    it("calls onBack from the shell when Back is clicked", async () => {
      const user = userEvent.setup();
      const onBack = vi.fn();

      render(
        <FirstRunChooseScreen
          repositories={["octo/repo"]}
          selectedRepository={null}
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
          selectedRepository="octo/stale-repo"
          loading
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByTestId("first-run-repositories-loading")).toBeInTheDocument();
      expect(screen.queryByRole("option", { name: /octo\/stale-repo/ })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: copy.useOrganization("octo") })).not.toBeInTheDocument();
      expect(screen.getByRole("status")).toHaveTextContent(copy.loading);
      expect(screen.getByTestId("first-run-repositories-spinner")).toBeInTheDocument();
    });

    it("shows new organizations while repositories synchronize", () => {
      const props = {
        selectedRepository: null,
        synchronizing: true,
        onSelectRepository: vi.fn(),
        onGrantAccess: vi.fn(),
        onContinue: vi.fn(),
      };
      const view = render(<FirstRunChooseScreen {...props} repositories={[]} />);

      const loading = screen.getByRole("status", { name: copy.loadingOrganizations });
      expect(loading.children).toHaveLength(2);
      expect(screen.queryByTestId("first-run-repositories-empty")).not.toBeInTheDocument();
      expect(
        within(screen.getByTestId("first-run-step-organization-header")).getByTestId(
          "first-run-repositories-synchronizing",
        ),
      ).toHaveTextContent(copy.synchronizingOrganizations);

      view.rerender(<FirstRunChooseScreen {...props} repositories={["acme/api"]} />);
      expect(screen.getByRole("button", { name: copy.useOrganization("acme") })).toBeInTheDocument();
      expect(screen.getByRole("status", { name: copy.loadingOrganizations }).children).toHaveLength(1);

      view.rerender(<FirstRunChooseScreen {...props} repositories={["acme/api", "octo/web"]} />);
      expect(screen.getByRole("button", { name: copy.useOrganization("acme") })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: copy.useOrganization("octo") })).toBeInTheDocument();

      view.rerender(<FirstRunChooseScreen {...props} synchronizing={false} repositories={["acme/api", "octo/web"]} />);
      expect(screen.queryByRole("status", { name: copy.loadingOrganizations })).not.toBeInTheDocument();
    });

    it("explains that only organizations with a writable repository appear", () => {
      render(
        <FirstRunChooseScreen
          repositories={["octo/repo"]}
          selectedRepository={null}
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByText(copy.organizationWriteAccessHint)).toBeInTheDocument();
    });
  });

  describe("repository step", () => {
    it("shows only the repositories of the chosen organization", async () => {
      const user = userEvent.setup();

      render(
        <FirstRunChooseScreen
          repositories={["forestileao/test", "superplanehq/superplane", "forestileao/instabot"]}
          selectedRepository={null}
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      await chooseOrganization(user, "forestileao");

      expect(screen.getByRole("heading", { name: copy.headline })).toBeInTheDocument();
      expect(screen.getByRole("option", { name: /forestileao\/test/ })).toBeInTheDocument();
      expect(screen.getByRole("option", { name: /forestileao\/instabot/ })).toBeInTheDocument();
      expect(screen.queryByRole("option", { name: /superplanehq\/superplane/ })).not.toBeInTheDocument();
      expect(screen.getByText(FIRST_RUN_COPY.connect.stepOrganizationDone("forestileao"))).toBeInTheDocument();
    });

    it("opens the owner of a saved repository", () => {
      render(
        <FirstRunChooseScreen
          repositories={["acme/api", "octo/repo"]}
          selectedRepository="octo/repo"
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByRole("option", { name: /octo\/repo/ })).toBeInTheDocument();
      expect(screen.queryByRole("option", { name: /acme\/api/ })).not.toBeInTheDocument();
    });

    it("goes back to the organization step instead of leaving the screen", async () => {
      const user = userEvent.setup();
      const onBack = vi.fn();

      render(
        <FirstRunChooseScreen
          repositories={["acme/api", "octo/repo"]}
          selectedRepository={null}
          chrome={{ stepIndex: 2, onBack }}
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      await chooseOrganization(user, "octo");
      await user.click(screen.getByTestId("first-run-back"));

      expect(onBack).not.toHaveBeenCalled();
      expect(screen.getByRole("button", { name: copy.useOrganization("acme") })).toBeInTheDocument();
    });

    it("clears a saved repository that the new organization does not own", async () => {
      const user = userEvent.setup();
      const onClearRepository = vi.fn();

      render(
        <FirstRunChooseScreen
          repositories={["acme/api", "octo/repo"]}
          selectedRepository="octo/repo"
          chrome={{ stepIndex: 2, onBack: vi.fn() }}
          onSelectRepository={vi.fn()}
          onClearRepository={onClearRepository}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      await user.click(screen.getByTestId("first-run-back"));
      await chooseOrganization(user, "octo");
      expect(onClearRepository).not.toHaveBeenCalled();

      await user.click(screen.getByTestId("first-run-back"));
      await chooseOrganization(user, "acme");
      expect(onClearRepository).toHaveBeenCalledTimes(1);
    });

    it("calls onGrantAccess when the grant access button is clicked", async () => {
      const user = userEvent.setup();
      const onGrantAccess = vi.fn();

      render(
        <FirstRunChooseScreen
          repositories={["octo/repo"]}
          selectedRepository="octo/repo"
          onSelectRepository={vi.fn()}
          onGrantAccess={onGrantAccess}
          onContinue={vi.fn()}
        />,
      );

      await user.click(screen.getByRole("button", { name: copy.grantAccess }));
      expect(onGrantAccess).toHaveBeenCalled();
    });

    it("disables continue until a repository is selected", async () => {
      const user = userEvent.setup();

      render(
        <FirstRunChooseScreen
          repositories={["octo/repo"]}
          selectedRepository={null}
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      await chooseOrganization(user, "octo");

      const continueButton = screen.getByTestId("first-run-continue-to-tickets");
      expect(continueButton).toBeDisabled();
      expect(continueButton).toHaveTextContent(copy.continue);
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
      expect(continueButton).toHaveTextContent(copy.continueReady);
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

      expect(screen.getByTestId("first-run-continue-to-tickets")).toHaveTextContent(copy.saving);
      expect(screen.getByRole("option", { name: /octo\/repo/ })).toBeDisabled();
      expect(screen.getByRole("button", { name: copy.grantAccess })).toBeDisabled();
    });

    it("shows the finished GitHub steps above the repository picker on the stepper card", () => {
      render(
        <FirstRunChooseScreen
          repositories={["puppies-inc/app"]}
          selectedRepository="puppies-inc/app"
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByTestId("first-run-github-stepper")).toBeInTheDocument();
      expect(screen.getByText(FIRST_RUN_COPY.connect.connectGitHub)).toBeInTheDocument();
      expect(screen.getAllByTestId("first-run-step-done")).toHaveLength(2);
      const repositoryList = screen.getByRole("listbox");
      expect(repositoryList).toHaveClass("max-h-48");
      expect(screen.getByRole("option", { name: /puppies-inc\/app/ })).toBeInTheDocument();
      expect(screen.getByTestId("first-run-continue-to-tickets")).toBeInTheDocument();
      expect(screen.getByTestId("first-run-content")).toHaveClass("py-16");
    });

    it("keeps repositories available while more repositories synchronize", () => {
      render(
        <FirstRunChooseScreen
          repositories={["octo/repo"]}
          selectedRepository="octo/repo"
          synchronizing
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByRole("option", { name: /octo\/repo/ })).toBeInTheDocument();
      const repositoryHeader = screen.getByTestId("first-run-step-repository-header");
      const status = within(repositoryHeader).getByTestId("first-run-repositories-synchronizing");
      expect(status).toHaveTextContent(copy.synchronizing);
      expect(status).toHaveClass("sp-ai-thinking");
      expect(screen.getByRole("status", { name: copy.loadingRepositories })).toBeInTheDocument();
    });

    it("explains that only writable repositories appear", () => {
      render(
        <FirstRunChooseScreen
          repositories={["octo/repo"]}
          selectedRepository="octo/repo"
          onSelectRepository={vi.fn()}
          onGrantAccess={vi.fn()}
          onContinue={vi.fn()}
        />,
      );

      expect(screen.getByText(copy.writeAccessHint)).toBeInTheDocument();
      expect(screen.queryByRole("status", { name: copy.loadingRepositories })).not.toBeInTheDocument();
    });
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
    expect(switchButton).toHaveTextContent(copy.switchAccount);
    expect(switchButton).toHaveClass("text-muted-foreground");

    await user.click(switchButton);
    await user.click(screen.getByRole("menuitemradio", { name: "forestigamer" }));
    expect(onSelectGitHubIdentity).toHaveBeenCalledWith("202");

    await user.click(switchButton);
    await user.click(screen.getByRole("menuitem", { name: copy.connectAnotherAccount }));
    expect(onConnectAnotherGitHubAccount).toHaveBeenCalled();
  });
});
