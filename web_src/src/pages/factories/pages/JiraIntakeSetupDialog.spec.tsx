import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { JiraIntakeSetupDialog } from "./JiraIntakeSetupDialog";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: () => ({
    data: [
      {
        metadata: { id: "integration-1", name: "Acme Jira", integrationName: "jira" },
        status: { state: "ready" },
      },
    ],
    isLoading: false,
    refetch: vi.fn(),
  }),
  useAvailableIntegrations: () => ({ data: [{ name: "jira", label: "Jira" }] }),
  useCreateIntegration: () => ({ mutateAsync: vi.fn(), reset: vi.fn() }),
  useIntegrationResources: () => ({
    data: [
      { id: "ENG", name: "Engineering (ENG)" },
      { id: "OPS", name: "Operations (OPS)" },
    ],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: ({ open, onCreated }: { open: boolean; onCreated: (id: string) => void }) =>
    open ? (
      <button type="button" data-testid="finish-connect" onClick={() => onCreated("integration-new")}>
        Finish connect
      </button>
    ) : null,
}));

describe("JiraIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
  });

  it("creates the intake once a connection and a project are chosen", async () => {
    const user = userEvent.setup();
    render(<JiraIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    // A single ready connection is preselected, so Continue opens the
    // project step.
    await waitFor(() => expect(screen.getByRole("button", { name: "Continue" })).toBeEnabled());
    await user.click(screen.getByRole("button", { name: "Acme Jira" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));

    await user.click(screen.getByTestId("jira-project-ENG"));
    await user.click(screen.getByRole("button", { name: "Create intake" }));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_JIRA_ISSUES",
        integrationId: "integration-1",
        resourceId: "ENG",
      });
    });

    expect(screen.getByText("Jira intake is ready")).toBeInTheDocument();
  });

  it("filters the project list by name", async () => {
    const user = userEvent.setup();
    render(<JiraIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Continue" }));
    await user.type(screen.getByLabelText("Search projects"), "oper");

    expect(screen.queryByTestId("jira-project-ENG")).not.toBeInTheDocument();
    expect(screen.getByTestId("jira-project-OPS")).toBeInTheDocument();
  });

  it("returns to the connection step from the project step", async () => {
    const user = userEvent.setup();
    render(<JiraIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Continue" }));
    await user.click(screen.getByRole("button", { name: "Back to connection" }));

    expect(screen.getByRole("button", { name: "Continue" })).toBeInTheDocument();
    expect(screen.queryByTestId("jira-project-ENG")).not.toBeInTheDocument();
  });

  it("opens project selection as soon as a new site is connected", async () => {
    const user = userEvent.setup();
    render(<JiraIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Connect another site" }));
    await user.click(screen.getByTestId("finish-connect"));

    expect(screen.getByTestId("jira-project-ENG")).toBeInTheDocument();
  });
});
