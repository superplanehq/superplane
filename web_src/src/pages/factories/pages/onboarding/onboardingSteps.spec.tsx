import type { OrganizationsIntegration } from "@/api-client";
import { render, renderHook, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { IntegrationInstanceSummary } from "@/pages/home/homeIntegrationStatus";
import { SetupSections } from "./OnboardingWireframe";
import { VcsStep } from "./onboardingSteps";
import { useOnboardingSetupState } from "./useOnboardingSetupState";

function githubConnection(id: string, name: string): OrganizationsIntegration {
  return {
    metadata: { id, name, integrationName: "github" },
    status: { state: "ready" },
  } as OrganizationsIntegration;
}

function githubIntegrations(...connections: OrganizationsIntegration[]): IntegrationInstanceSummary {
  return {
    name: "github",
    allInstances: connections,
    readyInstances: connections,
  };
}

describe("VcsStep", () => {
  it("lets the user choose an existing GitHub connection", async () => {
    const user = userEvent.setup();
    const onSelectConnection = vi.fn();

    render(
      <VcsStep
        github={githubIntegrations(
          githubConnection("github-1", "Product GitHub"),
          githubConnection("github-2", "Platform GitHub"),
        )}
        selectedConnectionId="github-1"
        onSelectConnection={onSelectConnection}
        onCreateConnection={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: /Platform GitHub/ }));

    expect(onSelectConnection).toHaveBeenCalledWith("github-2", "Platform GitHub");
  });

  it("lets the user create a new GitHub connection", async () => {
    const user = userEvent.setup();
    const onCreateConnection = vi.fn();

    render(
      <VcsStep
        github={githubIntegrations(githubConnection("github-1", "Product GitHub"))}
        selectedConnectionId="github-1"
        onSelectConnection={vi.fn()}
        onCreateConnection={onCreateConnection}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Connect new GitHub" }));

    expect(onCreateConnection).toHaveBeenCalledOnce();
  });

  it("shows a direct connect action when no GitHub connection exists", async () => {
    const user = userEvent.setup();
    const onCreateConnection = vi.fn();

    render(
      <VcsStep github={githubIntegrations()} onSelectConnection={vi.fn()} onCreateConnection={onCreateConnection} />,
    );

    await user.click(screen.getByRole("button", { name: "Connect GitHub" }));

    expect(onCreateConnection).toHaveBeenCalledOnce();
  });
});

describe("SetupSections start step", () => {
  function renderStartStep(saving: boolean) {
    const { result } = renderHook(() => useOnboardingSetupState("Payments Service"));

    render(
      <SetupSections
        setup={result.current}
        openSection="start"
        setOpenSection={vi.fn()}
        requestConnect={vi.fn()}
        createVcsConnection={vi.fn()}
        selectVcsConnection={vi.fn()}
        githubConnections={githubIntegrations()}
        onFinish={vi.fn()}
        saving={saving}
      />,
    );
  }

  it("shows the create action with no spinner while idle", () => {
    renderStartStep(false);

    const button = screen.getByTestId("workspace-setup-continue");
    expect(button).toHaveTextContent("Create work order");
    expect(button.querySelector("svg.animate-spin")).toBeNull();
  });

  it("shows a spinner and disables the button while the work order is created", () => {
    renderStartStep(true);

    const button = screen.getByTestId("workspace-setup-continue");
    expect(button).toHaveTextContent("Creating work order...");
    expect(button).toBeDisabled();
    expect(button.querySelector("svg.animate-spin")).not.toBeNull();
  });
});
