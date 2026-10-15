import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "bun:test";

import { FirstRunBitbucketForgeScreen } from "./FirstRunBitbucketForgeScreen";

describe("FirstRunBitbucketForgeScreen", () => {
  it("chooses a workspace before a repository and clears the selection when switching workspaces", async () => {
    const user = userEvent.setup();
    function WorkspaceChoice() {
      const [repository, setRepository] = useState<string | null>(null);
      return (
        <FirstRunBitbucketForgeScreen
          phase="choose"
          connectHref=""
          repositories={["acme-team/api", "other-team/web"]}
          installedWorkspaces={["empty-team"]}
          selectedRepository={repository}
          chrome={{ stepIndex: 2, onBack: vi.fn() }}
          onGrantAccess={vi.fn()}
          onSelectRepository={setRepository}
          onClearRepository={() => setRepository(null)}
          onContinue={vi.fn()}
        />
      );
    }
    render(<WorkspaceChoice />);

    expect(screen.getByRole("heading", { name: "Choose a Bitbucket workspace" })).toBeInTheDocument();
    expect(screen.queryByRole("option")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use empty-team" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Use acme-team" }));
    expect(screen.queryByRole("option", { name: /other-team\/web/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: /acme-team\/api/ }));
    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeEnabled();

    await user.click(screen.getByTestId("first-run-back"));
    await user.click(screen.getByRole("button", { name: "Use other-team" }));
    expect(screen.queryByRole("option", { name: /acme-team\/api/ })).not.toBeInTheDocument();
    expect(screen.getByRole("option", { name: /other-team\/web/ })).toBeInTheDocument();
    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeDisabled();
  });

  it("offers installation on another workspace when repositories are already available", async () => {
    const user = userEvent.setup();
    const onGrantAccess = vi.fn();
    render(
      <FirstRunBitbucketForgeScreen
        phase="choose"
        connectHref=""
        repositories={["acme-team/api"]}
        selectedRepository="acme-team/api"
        onGrantAccess={onGrantAccess}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Install on another workspace" }));

    expect(onGrantAccess).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("option", { name: /acme-team\/api/ })).toBeInTheDocument();
    expect(screen.getByText("You need write access to a repository to see it here.")).toBeInTheDocument();
    expect(screen.getByText("Do not see your repository?")).toBeInTheDocument();
  });

  it("shows the workspace hint with a subtle install link before a workspace is chosen", async () => {
    render(
      <FirstRunBitbucketForgeScreen
        phase="choose"
        connectHref=""
        repositories={["acme-team/api"]}
        installedWorkspaces={["acme-team"]}
        selectedRepository={null}
        onGrantAccess={vi.fn()}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByText("You need write access to at least one repository in a workspace.")).toBeInTheDocument();
    expect(screen.getByText("Do not see your Bitbucket workspace?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Install on another workspace" })).toHaveClass("underline");
  });

  it("explains a failed install-link copy and keeps the link selectable", async () => {
    const user = userEvent.setup();
    const installUrl = "https://bitbucket.org/site/addons/authorize?addon_key=superplane";
    const onCopyInstallLink = vi
      .fn<() => Promise<void>>()
      .mockRejectedValueOnce(new Error("denied"))
      .mockResolvedValueOnce(undefined);
    render(
      <FirstRunBitbucketForgeScreen
        phase="grant"
        connectHref=""
        repositories={[]}
        installedWorkspaces={[]}
        selectedRepository={null}
        installUrl={installUrl}
        onCopyInstallLink={onCopyInstallLink}
        chrome={{ stepIndex: 2, onBack: vi.fn() }}
        onGrantAccess={vi.fn()}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    const copyButton = screen.getByRole("button", { name: "Copy install link" });
    expect(copyButton).toHaveAttribute("data-slot", "button");
    await user.click(copyButton);

    expect(await screen.findByRole("alert")).toHaveTextContent("SuperPlane could not copy the install link.");
    expect(screen.getByText("Select the link below, or try again.")).toBeInTheDocument();
    const link = screen.getByRole("textbox", { name: "Install link" });
    expect(link).toHaveValue(installUrl);
    expect(link).toHaveAttribute("readonly");

    await user.click(copyButton);
    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("offers Check again after the installation attempt times out", async () => {
    const user = userEvent.setup();
    const onRetryLookup = vi.fn();
    render(
      <FirstRunBitbucketForgeScreen
        phase="grant"
        connectHref=""
        repositories={[]}
        installedWorkspaces={[]}
        selectedRepository={null}
        attemptTimedOut
        chrome={{ stepIndex: 2, onBack: vi.fn() }}
        onGrantAccess={vi.fn()}
        onRetryLookup={onRetryLookup}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByText(/SuperPlane did not detect a new workspace/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Check again" }));
    expect(onRetryLookup).toHaveBeenCalledTimes(1);
  });

  it("makes the Bitbucket wait obvious once the installation starts", async () => {
    render(
      <FirstRunBitbucketForgeScreen
        phase="grant"
        connectHref=""
        repositories={[]}
        installedWorkspaces={[]}
        selectedRepository={null}
        attemptActive
        chrome={{ stepIndex: 2, onBack: vi.fn() }}
        onGrantAccess={vi.fn()}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-bitbucket-waiting")).toHaveTextContent("Checking for the new workspace…");
    expect(screen.getByTestId("first-run-bitbucket-waiting")).toHaveTextContent(
      "This page updates automatically when Bitbucket confirms the installation.",
    );
  });

  it("shows the checking spinner while a second installation is in flight", async () => {
    render(
      <FirstRunBitbucketForgeScreen
        phase="choose"
        connectHref=""
        repositories={["acme-team/api"]}
        selectedRepository="acme-team/api"
        attemptActive
        onGrantAccess={vi.fn()}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByRole("status")).toHaveTextContent("Checking for the new workspace…");
    expect(screen.getByRole("status").querySelector("svg")).toHaveClass("animate-spin");
    expect(screen.getByRole("option", { name: /acme-team\/api/ })).toBeInTheDocument();
  });
});
